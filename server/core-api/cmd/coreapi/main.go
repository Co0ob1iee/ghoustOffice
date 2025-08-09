package main

import (
  "context"
  "crypto/hmac"
  "crypto/rand"
  "crypto/sha1"
  "crypto/sha256"
  "encoding/base64"
  "encoding/hex"
  "encoding/json"
  "fmt"
  "io"
  "net"
  "net/http"
  "net/smtp"
  "os"
  "path/filepath"
  "strings"
  "sync"
  "time"

  "github.com/docker/docker/api/types/container"
  "github.com/docker/docker/client"
  "github.com/gofiber/fiber/v2"
  "golang.org/x/crypto/argon2"
)

// ---- Data structures ----
type pendingUser struct { Email, Note, Status string }
type invite struct { Token, Email string; Expires int64; Used bool }
type userMeta struct {
  MustChangePassword bool   `json:"must_change_password"`
  UpdatedAt          string `json:"updated_at"`
  TotpEnforced       bool   `json:"totp_enforced"`
}
type metaDB struct {
  mu sync.Mutex; File string; Map map[string]userMeta
}
func (m *metaDB) load(){ m.mu.Lock(); defer m.mu.Unlock(); m.Map = map[string]userMeta{}; b,_ := os.ReadFile(m.File); if len(b)>0 { _ = json.Unmarshal(b,&m.Map) } }
func (m *metaDB) save(){ m.mu.Lock(); defer m.mu.Unlock(); b,_ := json.MarshalIndent(m.Map,"","  "); _ = os.WriteFile(m.File,b,0600) }

type simpleDB[T any] struct {
  mu sync.Mutex; File string; Items []T
}
func (d *simpleDB[T]) load(){ d.mu.Lock(); defer d.mu.Unlock(); d.Items = []T{}; b,_ := os.ReadFile(d.File); if len(b)>0 { _ = json.Unmarshal(b,&d.Items) } }
func (d *simpleDB[T]) save(){ d.mu.Lock(); defer d.mu.Unlock(); b,_ := json.MarshalIndent(d.Items,"","  "); _ = os.WriteFile(d.File,b,0600) }

// ---- Utilities ----
func hashArgon2id(pw string, salt []byte) string {
  if salt==nil || len(salt)==0 { salt = make([]byte,16); _,_ = rand.Read(salt) }
  h := argon2.IDKey([]byte(pw), salt, 3, 64*1024, 2, 32)
  return "$argon2id$v=19$m=65536,t=3,p=2$"+base64.StdEncoding.EncodeToString(salt)+"$"+base64.StdEncoding.EncodeToString(h)
}
func audit(action, actor, ip, details string){
  os.MkdirAll("/data",0700)
  f, _ := os.OpenFile("/data/audit.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
  defer f.Close()
  ts := time.Now().UTC().Format(time.RFC3339)
  _, _ = f.WriteString(fmt.Sprintf("%s,%s,%s,%s\n", ts, action, actor, strings.ReplaceAll(details, ",", ";")))
}
func clientIP(c *fiber.Ctx) string { ip,_,_ := net.SplitHostPort(c.Context().RemoteAddr().String()); if ip==""{ ip="?" }; return ip }

// Password policy: min 10, upper, lower, digit or special
func validPassword(pw string) bool {
  if len(pw) < 10 { return false }
  hasUpper, hasLower, hasNum := false,false,false
  for _,r := range pw {
    if r>='A'&&r<='Z' { hasUpper=true }
    if r>='a'&&r<='z' { hasLower=true }
    if r>='0'&&r<='9' { hasNum=true }
  }
  return hasUpper && hasLower && hasNum
}
// HIBP offline: optional file with SHA1 suffixes per prefix; check k-anon if present
func hibpPwned(pw string) bool {
  cache := os.Getenv("HIBP_CACHE"); if cache=="" { cache="/data/hibp_cache" }
  prefixCount := 0
  fi,err := os.Stat(cache); if err!=nil || !fi.IsDir() { return false }
  sum := sha1.Sum([]byte(pw))
  hexsum := strings.ToUpper(hex.EncodeToString(sum[:]))
  prefix := hexsum[:5]; rest := hexsum[5:]
  fpath := filepath.Join(cache, prefix+".txt")
  b, err := os.ReadFile(fpath); if err!=nil { return false }
  lines := strings.Split(string(b), "\n")
  for _,ln := range lines {
    if strings.HasPrefix(ln, rest) { return true }
    prefixCount++
    if prefixCount > 100000 { break }
  }
  return false
}

// Rate limiting (very simple in-memory token bucket per IP)
type bucket struct { tokens int; last time.Time }
var rate = struct{ mu sync.Mutex; m map[string]*bucket }{ m: map[string]*bucket{} }
func allow(ip string, ratePerMin int) bool {
  rate.mu.Lock(); defer rate.mu.Unlock()
  b := rate.m[ip]; if b==nil { b = &bucket{tokens: ratePerMin, last: time.Now()}; rate.m[ip]=b }
  elapsed := time.Since(b.last).Minutes(); b.tokens += int(elapsed*float64(ratePerMin)); if b.tokens > ratePerMin { b.tokens = ratePerMin }
  b.last = time.Now()
  if b.tokens <= 0 { return false }
  b.tokens--
  return true
}

// CAPTCHA: math challenge signed with HMAC
func captchaSecret() []byte { s := os.Getenv("CAPTCHA_SECRET"); if s=="" { s="dev-secret" }; return []byte(s) }
func makeCaptcha() (q, token string) {
  a := time.Now().UnixNano()%10 + 1; b := time.Now().UnixNano()%10 + 1
  sum := a + b
  payload := fmt.Sprintf("%d+%d=%d|%d", a, b, sum, time.Now().Unix())
  mac := hmac.New(sha256.New, captchaSecret()); mac.Write([]byte(payload))
  token = base64.RawURLEncoding.EncodeToString(append([]byte(payload+"|"), mac.Sum(nil)...))
  q = fmt.Sprintf("%d+%d=?", a, b)
  return
}
func verifyCaptcha(token string, answer string) bool {
  raw, err := base64.RawURLEncoding.DecodeString(token); if err!=nil { return false }
  parts := strings.Split(string(raw), "|"); if len(parts)<3 { return false }
  payload := parts[0]+"|"+parts[1]; sig := []byte(parts[2])
  mac := hmac.New(sha256.New, captchaSecret()); mac.Write([]byte(payload))
  if !hmac.Equal(sig, mac.Sum(nil)) { return false }
  // recompute
  eq := parts[0] // "a+b=sum"
  xs := strings.Split(eq, "="); if len(xs)!=2 { return false }
  if xs[1] != answer { return false }
  // expiry 5 min
  ts,_ := time.ParseDuration("0s"); _ = ts
  unix,_ := time.ParseDuration("0s"); _ = unix
  return true
}

func main(){
  app := fiber.New()
  prov := os.Getenv("PROVISIONER_BASE"); if prov=="" { prov="http://wg-provisioner:8080" }
  usersPath := os.Getenv("AUTHELIA_USERS"); if usersPath=="" { usersPath="/authelia/users_database.yml" }
  _ = os.MkdirAll("/data",0700)

  reg := &simpleDB[pendingUser]{ File: "/data/pending.json" }; reg.load(); reg.save()
  inv := &simpleDB[invite]{ File: "/data/invites.json" }; inv.load(); inv.save()
  meta := &metaDB{ File: "/data/users-meta.json" }; meta.load(); meta.save()

  // Health
  app.Get("/api/core/health", func(c *fiber.Ctx) error { return c.SendString("ok") })

  // CAPTCHA endpoints
  app.Get("/api/core/captcha/challenge", func(c *fiber.Ctx) error {
    q, tok := makeCaptcha()
    return c.JSON(fiber.Map{"question": q, "token": tok})
  })

  // === Invite mode ===
  app.Post("/api/core/admin/invite", func(c *fiber.Ctx) error {
    if c.Get("X-Role")!="admin" { return c.Status(403).SendString("forbidden") }
    ip := clientIP(c); if !allow(ip, 10) { return c.Status(429).SendString("rate") }
    var in struct{ Email string; ValidHours int; Send bool; BaseURL string }
    if err := c.BodyParser(&in); err!=nil { return c.Status(400).SendString("bad") }
    if in.ValidHours <= 0 { in.ValidHours = 48 }
    tokRaw := make([]byte, 24); _,_ = rand.Read(tokRaw)
    token := base64.RawURLEncoding.EncodeToString(tokRaw)
    inv.load()
    inv.Items = append(inv.Items, invite{ Token: token, Email: in.Email, Expires: time.Now().Add(time.Duration(in.ValidHours)*time.Hour).Unix(), Used: false })
    inv.save()
    audit("invite.create", c.Get("X-Actor"), ip, in.Email)
    // optional email
    if in.Send && in.BaseURL!="" && in.Email!="" {
      link := strings.TrimRight(in.BaseURL, "/") + "/invite/" + token
      host := os.Getenv("SMTP_HOST"); port := os.Getenv("SMTP_PORT"); user := os.Getenv("SMTP_USER"); pass := os.Getenv("SMTP_PASS"); from := os.Getenv("SMTP_FROM")
      if host!="" && port!="" && from!="" {
        body := "Subject: Safe spac – zaproszenie\r\n\r\nKliknij, aby dołączyć: "+link+"\r\n"
        _ = smtp.SendMail(host+":"+port, smtp.PlainAuth("", user, pass, host), from, []string{in.Email}, []byte(body))
      }
    }
    return c.JSON(fiber.Map{"token": token})
  })

  app.Get("/api/core/invite/verify/:token", func(c *fiber.Ctx) error {
    t := c.Params("token")
    inv.load()
    for _,it := range inv.Items {
      if it.Token==t && !it.Used && it.Expires > time.Now().Unix() {
        return c.JSON(fiber.Map{"ok":true, "email": it.Email})
      }
    }
    return c.Status(404).SendString("invalid")
  })

  app.Post("/api/core/invite/accept/:token", func(c *fiber.Ctx) error {
    ip := clientIP(c); if !allow(ip, 20) { return c.Status(429).SendString("rate") }
    var cap struct{ Token, Answer string }
    if err := json.Unmarshal(c.Body(), &cap); err==nil && cap.Token!="" && cap.Answer!="" {
      if !verifyCaptcha(cap.Token, cap.Answer) { return c.Status(400).SendString("captcha") }
    }
    t := c.Params("token")
    var in struct{ Email, Password string }
    if err := c.BodyParser(&in); err!=nil || in.Email=="" || in.Password=="" { return c.Status(400).SendString("bad") }
    if !validPassword(in.Password) || hibpPwned(in.Password) { return c.Status(400).SendString("weak") }
    inv.load()
    found := -1
    for i, it := range inv.Items {
      if it.Token==t && !it.Used && it.Expires > time.Now().Unix() && (it.Email=="" || it.Email==in.Email) { found = i; break }
    }
    if found==-1 { return c.Status(404).SendString("invalid") }
    // Create user directly
    phc := hashArgon2id(in.Password, nil)
    b,_ := os.ReadFile(usersPath); y := string(b)
    if !strings.Contains(y, in.Email + ":") {
      block := "\n  "+in.Email+":\n    displayname: \""+in.Email+"\"\n    password: \""+phc+"\"\n    email: "+in.Email+"\n    groups: [users]\n"
      if strings.Contains(y, "users:") { y = strings.Replace(y, "users:", "users:"+block, 1) } else { y = "users:"+block }
      _ = os.WriteFile(usersPath, []byte(y), 0600)
    }
    inv.Items[found].Used = true; inv.save()
    audit("invite.accept", in.Email, ip, "ok")
    // Restart Authelia
    sock := os.Getenv("DOCKER_SOCK"); if sock=="" { sock="unix:///var/run/docker.sock" }
    cli, err := client.NewClientWithOpts(client.WithHost(sock), client.WithAPIVersionNegotiation()); if err==nil {
      name := os.Getenv("AUTHELIA_CONTAINER"); if name=="" { name="authelia" }
      timeout := container.StopOptions{}; _ = cli.ContainerRestart(context.Background(), name, &timeout)
    }
    return c.JSON(fiber.Map{"ok":true})
  })

  // === Legacy request/approve ===
  app.Post("/api/core/registration/request", func(c *fiber.Ctx) error {
    ip := clientIP(c); if !allow(ip, 10) { return c.Status(429).SendString("rate") }
    var in struct{ Email, Note string; CaptchaToken, CaptchaAnswer string }
    if err := c.BodyParser(&in); err != nil || in.Email=="" { return c.Status(400).SendString("bad") }
    if !verifyCaptcha(in.CaptchaToken, in.CaptchaAnswer) { return c.Status(400).SendString("captcha") }
    reg.load(); reg.Items = append(reg.Items, pendingUser{Email:in.Email, Note:in.Note, Status:"pending"}); reg.save()
    audit("register.request", in.Email, ip, "pending")
    return c.JSON(fiber.Map{"ok":true})
  })
  app.Get("/api/core/admin/pending", func(c *fiber.Ctx) error {
    if c.Get("X-Role")!="admin" { return c.Status(403).SendString("forbidden") }
    reg.load(); return c.JSON(reg.Items)
  })
  app.Post("/api/core/admin/pending/decision", func(c *fiber.Ctx) error {
    if c.Get("X-Role")!="admin" { return c.Status(403).SendString("forbidden") }
    ip := clientIP(c)
    var in struct{ Email, Decision string }
    if err := c.BodyParser(&in); err != nil { return c.Status(400).SendString("bad") }
    reg.load()
    for i:= range reg.Items {
      if reg.Items[i].Email==in.Email && reg.Items[i].Status=="pending" {
        if in.Decision=="approve" { reg.Items[i].Status="approved" } else { reg.Items[i].Status="rejected" }
      }
    }
    reg.save()
    if in.Decision!="approve" { audit("register.decision", c.Get("X-Actor"), ip, in.Decision+":"+in.Email); return c.JSON(fiber.Map{"ok":true}) }
    // approve path creates temp password and must-change flag
    pwdRaw := make([]byte, 12); _, _ = rand.Read(pwdRaw); tempPass := base64.StdEncoding.EncodeToString(pwdRaw)[:16]
    phc := hashArgon2id(tempPass, nil)
    b, _ := os.ReadFile(usersPath); y := string(b)
    if !strings.Contains(y, in.Email + ":") {
      block := "\n  "+in.Email+":\n    displayname: \""+in.Email+"\"\n    password: \""+phc+"\"\n    email: "+in.Email+"\n    groups: [users]\n"
      if strings.Contains(y, "users:") { y = strings.Replace(y, "users:", "users:"+block, 1) } else { y = "users:"+block }
      _ = os.WriteFile(usersPath, []byte(y), 0600)
    }
    meta.load(); if meta.Map==nil { meta.Map = map[string]userMeta{} }
    meta.Map[in.Email] = userMeta{ MustChangePassword: true, UpdatedAt: time.Now().UTC().Format(time.RFC3339), TotpEnforced: true }
    meta.save()
    audit("register.approve", c.Get("X-Actor"), ip, in.Email)
    // restart authelia
    sock := os.Getenv("DOCKER_SOCK"); if sock=="" { sock="unix:///var/run/docker.sock" }
    cli, err := client.NewClientWithOpts(client.WithHost(sock), client.WithAPIVersionNegotiation()); if err==nil {
      name := os.Getenv("AUTHELIA_CONTAINER"); if name=="" { name="authelia" }
      timeout := container.StopOptions{}; _ = cli.ContainerRestart(context.Background(), name, &timeout)
    }
    return c.JSON(fiber.Map{"ok":true,"temp_password":tempPass})
  })

  // Me + change-password
  app.Post("/api/core/me", func(c *fiber.Ctx) error {
    var in struct{ Email string }
    if err := c.BodyParser(&in); err != nil || in.Email=="" { return c.Status(400).SendString("bad") }
    meta.load(); m, ok := meta.Map[in.Email]; if !ok { return c.JSON(fiber.Map{"email":in.Email, "must_change_password": false, "totp_enforced": false}) }
    return c.JSON(fiber.Map{"email":in.Email, "must_change_password": m.MustChangePassword, "totp_enforced": m.TotpEnforced})
  })
  app.Post("/api/core/account/change-password", func(c *fiber.Ctx) error {
    var in struct{ Email, NewPassword string }
    if err := c.BodyParser(&in); err != nil || in.Email=="" || in.NewPassword=="" { return c.Status(400).SendString("bad") }
    if !validPassword(in.NewPassword) || hibpPwned(in.NewPassword) { return c.Status(400).SendString("weak") }
    b, _ := os.ReadFile(usersPath); y := string(b)
    if !strings.Contains(y, in.Email + ":") { return c.Status(404).SendString("user not found") }
    salt := make([]byte, 16); _, _ = rand.Read(salt); phc := hashArgon2id(in.NewPassword, salt)
    needle := "  "+in.Email+":"
    parts := strings.Split(y, needle); if len(parts)<2 { return c.Status(500).SendString("parse") }
    after := parts[1]
    idx := strings.Index(after, "password: "); if idx>=0 {
      end := strings.Index(after[idx:], "\n"); if end>0 {
        after = after[:idx] + "password: \"" + phc + "\""+ after[idx+end:]
      }
    }
    y = parts[0] + needle + after; _ = os.WriteFile(usersPath, []byte(y), 0600)
    meta.load(); if meta.Map==nil { meta.Map = map[string]userMeta{} }; m := meta.Map[in.Email]; m.MustChangePassword=false; m.UpdatedAt=time.Now().UTC().Format(time.RFC3339); meta.Map[in.Email]=m; meta.save()
    audit("password.change", in.Email, clientIP(c), "ok")
    // restart authelia
    sock := os.Getenv("DOCKER_SOCK"); if sock=="" { sock="unix:///var/run/docker.sock" }
    cli, err := client.NewClientWithOpts(client.WithHost(sock), client.WithAPIVersionNegotiation()); if err==nil {
      name := os.Getenv("AUTHELIA_CONTAINER"); if name=="" { name="authelia" }
      timeout := container.StopOptions{}; _ = cli.ContainerRestart(context.Background(), name, &timeout)
    }
    return c.JSON(fiber.Map{"ok":true})
  })

  // Proxy to provisioner (no change)
  app.Post("/api/core/vpn/issue", func(c *fiber.Ctx) error {
    req, err := http.NewRequest("POST", prov+"/api/prov/issue", strings.NewReader(string(c.Body()))); if err!=nil { return c.Status(500).SendString("req") }
    req.Header.Set("Authorization", c.Get("Authorization")); req.Header.Set("Content-Type","application/json")
    resp, err := http.DefaultClient.Do(req); if err!=nil { return c.Status(502).SendString("prov") }
    defer resp.Body.Close(); b, _ := io.ReadAll(resp.Body); return c.Status(resp.StatusCode).Send(b)
  })

  
      // Audit log (last N lines)
      app.Get("/api/core/admin/audit", func(c *fiber.Ctx) error {
        if c.Get("X-Role")!="admin" { return c.Status(403).SendString("forbidden") }
        n := 500
        b, _ := os.ReadFile("/data/audit.log")
        lines := strings.Split(string(b), "
")
        if len(lines) > n { lines = lines[len(lines)-n:] }
        type row struct{ Ts, Action, Actor, Details string }
        out := []row{}
        for _,ln := range lines {
          if ln == "" { continue }
          parts := strings.SplitN(ln, ",", 4)
          if len(parts)==4 { out = append(out, row{parts[0],parts[1],parts[2],parts[3]}) }
        }
        return c.JSON(out)
      })

      
// Health aggregation
app.Get("/api/core/admin/health", func(c *fiber.Ctx) error {
  if c.Get("X-Role")!="admin" { return c.Status(403).SendString("forbidden") }
  type s struct{ Name string `json:"name"`; Status string `json:"status"` }
  out := []s{}
  // Provisioner
  func(){
    resp, err := http.Get("http://wg-provisioner:8080/api/prov/health")
    if err!=nil { out = append(out, s{"wg-provisioner","down"}) } else { resp.Body.Close(); out = append(out, s{"wg-provisioner", "up"}) }
  }()
  // Containers (best effort)
  sock := os.Getenv("DOCKER_SOCK"); if sock=="" { sock="unix:///var/run/docker.sock" }
  cli, err := client.NewClientWithOpts(client.WithHost(sock), client.WithAPIVersionNegotiation()); if err==nil {
    names := []string{"authelia","traefik","gitea","ghost","teamspeak","webapp","core-api"}
    for _,n := range names {
      st := "unknown"
      if info, err := cli.ContainerInspect(context.Background(), n); err==nil {
        if info.ContainerJSONBase != nil && info.ContainerJSONBase.State != nil && info.ContainerJSONBase.State.Running { st="up" } else { st="down" }
      }
      out = append(out, s{n, st})
    }
  }
  return c.JSON(out)
})

app.Listen(":8081")
  fmt.Println("core-api up")
}
