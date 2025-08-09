'use client'\nimport {Button} from '@/components/ui/button'\nimport {Input} from '@/components/ui/input'\nimport {Card} from '@/components/ui/card'
import {useEffect, useState} from 'react'
export default function ChangePassword(){
  const [email,setEmail]=useState(''); const [pwd,setPwd]=useState(''); const [msg,setMsg]=useState('')
  useEffect(()=>{
    try{
      const tok = window.localStorage.getItem('id_token')||''
      const payload = tok.split('.')[1]? JSON.parse(atob(tok.split('.')[1].replace(/-/g,'+').replace(/_/g,'/'))) : {}
      setEmail(payload.email||payload.preferred_username||'')
    }catch(e){}
  },[])
  const submit = async ()=>{
    if(pwd.length < 10){ setMsg('Hasło min. 10 znaków.'); return }
    const r = await fetch('/api/core/account/change-password',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email, NewPassword: pwd})})
    setMsg(r.ok? 'Hasło zmienione. Zaloguj ponownie.' : 'Błąd zmiany hasła')
  }
  return (<div className=" space-y">
    <h2 className="title">Zmiana hasła (wymagana)</h2>
    <p className="subtitle">Twoje konto zostało utworzone. Zmień hasło i włącz TOTP przy pierwszym logowaniu.</p>
    <div className="row"><label className="label">Email</label><Input className="input" value={email} readOnly style={{width:360}}/></div>
    <div className="row"><label className="label">Nowe hasło</label><Input className="input" type="password" value={pwd} onChange={e=>setPwd(e.target.value)} style={{width:360}}/></div>
    <div className="row"><Button className=" btn-ok" onClick={submit}>Zmień</Button></div>
    {msg && <div className="alert">{msg}</div>}
  </div></Card>)
}
