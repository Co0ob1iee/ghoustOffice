'use client'\nimport {Button} from '@/components/ui/button'\nimport {Input} from '@/components/ui/input'\nimport {Card} from '@/components/ui/card'
import {useState} from 'react'

export default function AppHome(){
  const [mode,setMode]=useState('normal')
  const [scope,setScope]=useState('split')
  const [cfg,setCfg]=useState('')
  async function issue(){
    const token = window.localStorage.getItem('access_token') || ''
    const res = await fetch('/api/core/vpn/issue', {
      method:'POST',
      headers:{'Authorization':'Bearer '+token,'Content-Type':'application/json'},
      body: JSON.stringify({ pubkey:'CLIENT_PUBKEY_PLACEHOLDER', mode, scope })
    })
    const data = await res.json()
    setCfg(data.config || JSON.stringify(data))
  }
  return (<div className="grid grid-2-1">
    <div className="">
      <h2>Panel użytkownika</h2>
      <p>Wydaj profil WireGuard (Normal/TOR × Split/Full)</p>
      <div>
        <label className="label">Tryb</label>
        <select className="select" value={mode} onChange={e=>setMode(e.target.value)}>
          <option value="normal">normal</option>
          <option value="tor">tor</option>
        </select>
        <label style={{marginLeft:12}}>Zakres: </label>
        <select className="select" value={scope} onChange={e=>setScope(e.target.value)}>
          <option value="split">split</option>
          <option value="full">full</option>
        </select>
        <Button className=" btn-ok" onClick={issue}>Wydaj profil</Button>
      
      // Check must-change-password
      async function mustChange(){
        try{
          const tok = window.localStorage.getItem('id_token')||''
          const payload = tok.split('.')[1]? JSON.parse(atob(tok.split('.')[1].replace(/-/g,'+').replace(/_/g,'/'))) : {}
          const r = await fetch('/api/core/me',{method:'POST',headers:{'Content-Type':'application/json'},body: JSON.stringify({email: payload.email||payload.preferred_username||''})})
          const j = await r.json()
          if(j.must_change_password){ window.location.href='/app/change-password' }
        }catch(e){}
      }
      mustChange()
    
</div>
      <pre style={{whiteSpace:'pre-wrap', marginTop:12}}>{cfg}</pre>
    
      // Check must-change-password
      async function mustChange(){
        try{
          const tok = window.localStorage.getItem('id_token')||''
          const payload = tok.split('.')[1]? JSON.parse(atob(tok.split('.')[1].replace(/-/g,'+').replace(/_/g,'/'))) : {}
          const r = await fetch('/api/core/me',{method:'POST',headers:{'Content-Type':'application/json'},body: JSON.stringify({email: payload.email||payload.preferred_username||''})})
          const j = await r.json()
          if(j.must_change_password){ window.location.href='/app/change-password' }
        }catch(e){}
      }
      mustChange()
    
</div>
    <div className="">
      <h3>Widget użytkownika</h3>
      <p>Status: zalogowany (placeholder)</p>
      <a href="/admin">Panel administracyjny</a>
    
      // Check must-change-password
      async function mustChange(){
        try{
          const tok = window.localStorage.getItem('id_token')||''
          const payload = tok.split('.')[1]? JSON.parse(atob(tok.split('.')[1].replace(/-/g,'+').replace(/_/g,'/'))) : {}
          const r = await fetch('/api/core/me',{method:'POST',headers:{'Content-Type':'application/json'},body: JSON.stringify({email: payload.email||payload.preferred_username||''})})
          const j = await r.json()
          if(j.must_change_password){ window.location.href='/app/change-password' }
        }catch(e){}
      }
      mustChange()
    
</div>
  
      // Check must-change-password
      async function mustChange(){
        try{
          const tok = window.localStorage.getItem('id_token')||''
          const payload = tok.split('.')[1]? JSON.parse(atob(tok.split('.')[1].replace(/-/g,'+').replace(/_/g,'/'))) : {}
          const r = await fetch('/api/core/me',{method:'POST',headers:{'Content-Type':'application/json'},body: JSON.stringify({email: payload.email||payload.preferred_username||''})})
          const j = await r.json()
          if(j.must_change_password){ window.location.href='/app/change-password' }
        }catch(e){}
      }
      mustChange()
    
</div></Card>)
}
