const {app, BrowserWindow, Tray, Menu, shell, ipcMain} = require('electron')
const path = require('path')
const http = require('http')
const crypto = require('crypto')
const fetch = (...args) => import('node-fetch').then(({default: fetch}) => fetch(...args))

let tray=null, win=null, tokenStore = { access_token:'', refresh_token:'', expires_at:0, issuer:'' }

function b64url(input){ return input.toString('base64').replace(/\+/g,'-').replace(/\//g,'_').replace(/=+$/,'') }
function pkce(){ const v = b64url(crypto.randomBytes(32)); const c = b64url(crypto.createHash('sha256').update(v).digest()); return {verifier:v, challenge:c} }

async function discover(issuer){
  const well = issuer.replace(/\/$/,'') + '/.well-known/openid-configuration'
  const res = await fetch(well); if(!res.ok) throw new Error('OIDC discovery failed')
  return res.json()
}

async function exchangeToken(issuer, token_endpoint, code, verifier, client_id='vpn-app', redirect_uri='http://127.0.0.1:53123/cb'){
  const params = new URLSearchParams()
  params.set('grant_type','authorization_code')
  params.set('code', code)
  params.set('client_id', client_id)
  params.set('redirect_uri', redirect_uri)
  params.set('code_verifier', verifier)
  const r = await fetch(token_endpoint, { method:'POST', headers:{'Content-Type':'application/x-www-form-urlencoded'}, body: params.toString() })
  if(!r.ok){ throw new Error('Token exchange failed: '+r.status) }
  return r.json()
}

function saveToken(t, issuer){
  tokenStore = {
    access_token: t.access_token || '',
    refresh_token: t.refresh_token || '',
    expires_at: Date.now() + (t.expires_in? t.expires_in*1000 : 3600*1000),
    issuer
  }
}

async function loginOIDC(issuer, audience='vpn-app'){
  const conf = await discover(issuer)
  const {verifier, challenge} = pkce()
  const state = b64url(crypto.randomBytes(16))
  const redirect_uri = 'http://127.0.0.1:53123/cb'
  const url = new URL(conf.authorization_endpoint)
  url.searchParams.set('client_id','vpn-app')
  url.searchParams.set('response_type','code')
  url.searchParams.set('redirect_uri', redirect_uri)
  url.searchParams.set('scope','openid profile email groups offline_access')
  url.searchParams.set('code_challenge', challenge)
  url.searchParams.set('code_challenge_method','S256')
  url.searchParams.set('state', state)
  // Open system browser
  await shell.openExternal(url.toString())

  // Local loopback server to catch callback
  const code = await new Promise((resolve, reject)=>{
    const srv = http.createServer((req,res)=>{
      if(req.url.startsWith('/cb')){
        const u = new URL('http://localhost'+req.url)
        const code = u.searchParams.get('code'); const st = u.searchParams.get('state')
        if(st !== state){ res.writeHead(400); res.end('State mismatch'); reject(new Error('state')); return }
        res.writeHead(200,{'Content-Type':'text/html'}); res.end('<h3>Login OK – wróć do aplikacji</h3>')
        srv.close(); resolve(code)
      } else { res.writeHead(404); res.end() }
    }).listen(53123,'127.0.0.1')
    setTimeout(()=>{ try{ srv.close() }catch(e){}; reject(new Error('timeout')) }, 3*60*1000)
  })

  const toks = await exchangeToken(issuer, conf.token_endpoint, code, verifier)
  saveToken(toks, issuer)
  return tokenStore
}

async function refreshToken(issuer){
  if(!tokenStore.refresh_token) return
  try{
    const well = issuer.replace(/\/$/,'') + '/.well-known/openid-configuration'
    const conf = await (await fetch(well)).json()
    const params = new URLSearchParams()
    params.set('grant_type','refresh_token')
    params.set('refresh_token', tokenStore.refresh_token)
    params.set('client_id','vpn-app')
    const r = await fetch(conf.token_endpoint, { method:'POST', headers:{'Content-Type':'application/x-www-form-urlencoded'}, body: params.toString() })
    if(!r.ok) return
    const t = await r.json()
    if(t.access_token){
      tokenStore.access_token = t.access_token
      if(t.refresh_token) tokenStore.refresh_token = t.refresh_token
      tokenStore.expires_at = Date.now() + (t.expires_in? t.expires_in*1000 : 3600*1000)
    }
  }catch(e){ /* ignore */ }
}

function getWindowHTML(){
  return `
    <body style="font-family: system-ui; background:#0b0f17; color:#e6eef8;">
      <h2 style="margin:16px">RBH VpnApp</h2>
      <div style="margin:16px">
        <label>Issuer:</label> <input id="issuer" value="https://portal.example.com/auth" style="width:360px">
        <label style="margin-left:8px">Mode:</label>
        <select id="mode"><option>normal</option><option>tor</option></select>
        <label style="margin-left:8px">Scope:</label>
        <select id="scope"><option>split</option><option>full</option></select>
        <button id="btnLogin">Zaloguj (OIDC)</button>
        <button id="btnIssue" disabled>Wydaj profil</button>
        <pre id="out" style="white-space:pre-wrap; margin-top:12px;"></pre>
      </div>
      <script>
        const btnLogin = document.getElementById('btnLogin')
        const btnIssue = document.getElementById('btnIssue')
        btnLogin.onclick = async ()=>{
          const issuer = document.getElementById('issuer').value
          const res = await window.electronAPI.oidcLogin(issuer)
          document.getElementById('out').innerText = 'Zalogowano. access_token...' 
          btnIssue.disabled = false
        }
        btnIssue.onclick = async ()=>{
          const issuer = document.getElementById('issuer').value
          const mode = document.getElementById('mode').value
          const scope = document.getElementById('scope').value
          const cfg = await window.electronAPI.issueProfile(issuer, mode, scope)
          document.getElementById('out').innerText = cfg
        }
      </script>
    </body>`
}

function createWindow(){
  win = new BrowserWindow({width:900,height:600,webPreferences:{nodeIntegration:false, contextIsolation:true,
    preload: path.join(__dirname,'preload.js')}})
  win.loadURL('data:text/html,'+encodeURIComponent(getWindowHTML()))
}

ipcMain.handle('oidc-login', async (event, issuer)=>{
  const t = await loginOIDC(issuer)
  return { ok:true }
})

ipcMain.handle('issue-profile', async (event, issuer, mode, scope)=>{
  // core-api proxied under same domain: https://<domain>/api/core/vpn/issue
  if(!tokenStore.access_token || tokenStore.issuer !== issuer) throw new Error('Brak tokenu lub inny issuer')
  const apiBase = issuer.replace(/\/auth\/?$/,'')
  const res = await fetch(apiBase+'/api/core/vpn/issue', {
    method:'POST',
    headers:{'Authorization':'Bearer '+tokenStore.access_token,'Content-Type':'application/json'},
    body: JSON.stringify({ pubkey:'CLIENT_PUBKEY_PLACEHOLDER', mode, scope })
  })
  const data = await res.json().catch(()=>({}))
  return data.config ? data.config : JSON.stringify(data)
})

app.whenReady().then(()=>{
  setInterval(async ()=>{ if(tokenStore.access_token && (tokenStore.expires_at - Date.now() < 60000)){ await refreshToken(tokenStore.issuer) } }, 10000);
  createWindow()
  tray = new Tray(path.join(__dirname,'..','build','icon.ico'))
  tray.setToolTip('RBH VpnApp')
  tray.setContextMenu(Menu.buildFromTemplate([
    {label:'Połącz (mock)', click:()=>{}},
    {label:'Rozłącz (mock)', click:()=>{}},
    {type:'separator'},
    {label:'Wyjście', click:()=>app.quit()}
  ]))
})

const {contextBridge} = require('electron')
