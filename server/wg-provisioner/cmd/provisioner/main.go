package main

import (
  "context"
  "encoding/json"
  "errors"
  "fmt"
  "net/netip"
  "os"
  "strings"
  "sync"

  "github.com/gofiber/fiber/v2"
  "github.com/lestrrat-go/jwx/v2/jwk"
  "github.com/lestrrat-go/jwx/v2/jwt"
)

type Req struct {
  PubKey string `json:"pubkey"`
  Mode   string `json:"mode"`   // normal|tor
  Scope  string `json:"scope"`  // full|split
  Label  string `json:"label,omitempty"`
}

type Resp struct {
  Config       string `json:"config"`
  TsSupported  bool   `json:"ts_supported"`
  TsURI        string `json:"ts_uri,omitempty"`
  ProfileName  string `json:"profile_name"`
  AssignedIP   string `json:"assigned_ip"`
  ServerPubKey string `json:"server_pubkey"`
}

type AllocDB struct {
  sync.Mutex
  File string
  IP   map[string]bool
}
func (a *AllocDB) Load(){ a.IP = map[string]bool{}; b,_ := os.ReadFile(a.File); if len(b)>0 { _=json.Unmarshal(b,&a.IP) } }
func (a *AllocDB) Save(){ b,_ := json.MarshalIndent(a.IP,"","  "); _=os.WriteFile(a.File,b,0600) }

func pickIP(cidr string, used map[string]bool) (string, error) {
  pfx, err := netip.ParsePrefix(cidr)
  if err != nil { return "", err }
  net := pfx.Masked().Addr().As4()
  for i:=10;i<250;i++ {
    ip := fmt.Sprintf("%d.%d.%d.%d", net[0],net[1],net[2],i)
    if !used[ip] { return ip,nil }
  }
  return "", errors.New("no free IPs")
}

func main(){
  app := fiber.New()
  issuer := os.Getenv("OIDC_ISSUER")
  aud := os.Getenv("OIDC_AUDIENCE")
  jwksURL := strings.TrimSuffix(issuer,"/") + "/jwks.json"
  jwks, err := jwk.Fetch(context.Background(), jwksURL)
  if err != nil { panic(err) }

  wg0Conf := os.Getenv("WG0_CONF")
  wg1Conf := os.Getenv("WG1_CONF")
  wg0Net := os.Getenv("WG0_NET")
  wg1Net := os.Getenv("WG1_NET")
  wg0Endpoint := os.Getenv("WG0_ENDPOINT")
  wg1Endpoint := os.Getenv("WG1_ENDPOINT")
  serverPubWg0 := os.Getenv("SERVER_PUB_WG0")
  serverPubWg1 := os.Getenv("SERVER_PUB_WG1")
  tsHost := os.Getenv("TS_HOST")

  db := &AllocDB{File: "/etc/wireguard/alloc.db"}; db.Load()

  auth := func(c *fiber.Ctx) (*jwt.Token, error) {
    h := c.Get("Authorization"); if !strings.HasPrefix(h,"Bearer ") { return nil, errors.New("no bearer") }
    raw := strings.TrimPrefix(h,"Bearer ")
    t, err := jwt.Parse([]byte(raw), jwt.WithKeySet(jwks), jwt.WithValidate(true))
    if err != nil { return nil, err }
    if iss,_ := t.Issuer(); iss != issuer { return nil, errors.New("bad iss") }
    if aud != "" && !t.Audience().Contains(aud) { return nil, errors.New("bad aud") }
    gr, ok := t.Get("groups"); if !ok { return nil, errors.New("no groups") }
    okUser := false
    switch v := gr.(type) {
    case []any:
      for _,g := range v { if gs,ok := g.(string); ok && (gs=="users" || gs=="admins") { okUser=true } }
    case []string:
      for _,g := range v { if g=="users" || g=="admins" { okUser=true } }
    default:
      okUser = false
    }
    if !okUser { return nil, errors.New("forbidden") }
        if amrClaim, ok := t.Get("amr"); ok {
          switch vv := amrClaim.(type) {
          case []any:
            found := false; for _,x := range vv { if s,ok := x.(string); ok && s=="totp" { found=true } }
            if !found { return nil, errors.New("totp_required") }
          case []string:
            found := false; for _,x := range vv { if x=="totp" { found=true } }
            if !found { return nil, errors.New("totp_required") }
          }
        }
    return &t, nil
  }

  app.Post("/api/prov/issue", func(c *fiber.Ctx) error {
    if _, err := auth(c); err != nil { return c.Status(401).SendString(err.Error()) }
    var req Req
    if err := c.BodyParser(&req); err != nil { return c.Status(400).SendString("bad json") }
    if req.Mode!="normal" && req.Mode!="tor" { return c.Status(400).SendString("mode") }
    if req.Scope!="full" && req.Scope!="split" { return c.Status(400).SendString("scope") }

    var confPath, endpoint, netCIDR, serverPub string
    tsSupported := false
    if req.Mode=="normal" {
      confPath, endpoint, netCIDR, serverPub = wg0Conf, wg0Endpoint, wg0Net, serverPubWg0
      tsSupported = true
    } else {
      confPath, endpoint, netCIDR, serverPub = wg1Conf, wg1Endpoint, wg1Net, serverPubWg1
      tsSupported = false
    }

    db.Lock(); defer db.Unlock(); db.Load()
    ip, err := pickIP(netCIDR, db.IP); if err != nil { return c.Status(500).SendString("no ips") }
    db.IP[ip] = true; db.Save()

    peer := fmt.Sprintf("
[Peer]
# %s
PublicKey = %s
AllowedIPs = %s/32
", req.Label, req.PubKey, ip)
    f, err := os.OpenFile(confPath, os.O_APPEND|os.O_WRONLY, 0600); if err != nil { return c.Status(500).SendString("conf open") }
    defer f.Close()
    if _, err := f.WriteString(peer); err != nil { return c.Status(500).SendString("conf write") }

    allowed := "0.0.0.0/0, ::/0"
    if req.Scope=="split" { allowed = strings.Split(netCIDR,"/")[0] + "/24" }
    cfg := fmt.Sprintf(`[Interface]
PrivateKey =
Address = %s/32
DNS = %s

[Peer]
PublicKey = %s
AllowedIPs = %s
Endpoint = %s
PersistentKeepalive = 25
`, ip, strings.Split(netCIDR,"/")[0], serverPub, allowed, endpoint)

    resp := Resp{
      Config: cfg,
      TsSupported: tsSupported && req.Scope=="split",
      TsURI: fmt.Sprintf("tsserver://%s?port=9987", tsHost),
      ProfileName: fmt.Sprintf("rbh-%s-%s", req.Mode, req.Scope),
      AssignedIP: ip, ServerPubKey: serverPub,
    }
    return c.JSON(resp)
  })

  app.Get("/api/prov/health", func(c *fiber.Ctx) error { return c.SendString("ok") })
  _ = app.Listen(":8080")
}
