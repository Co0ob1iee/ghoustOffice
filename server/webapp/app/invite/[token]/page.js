'use client'
import {useEffect, useState} from 'react'
export default function Invite({ params }){
  const { token } = params
  const [email,setEmail]=useState('')
  const [pwd,setPwd]=useState('')
  const [msg,setMsg]=useState('Weryfikuję zaproszenie...'); const [capQ,setCapQ]=useState(''); const [capT,setCapT]=useState(''); const [capA,setCapA]=useState('')
  const [ok,setOk]=useState(false)
  useEffect(()=>{
    fetch('/api/core/invite/verify/'+token).then(r=>r.json()).then(j=>{
      setOk(true); setMsg('Zaproszenie OK')
      if(j.email) setEmail(j.email)
    }).catch(()=>{ setMsg('Zaproszenie nieważne lub zużyte'); setOk(false)})
  },[token])
  const accept = async ()=>{
    if(pwd.length<10){ setMsg('Hasło min. 10 znaków.'); return }
    const r = await fetch('/api/core/invite/accept/'+token,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email:email, password: pwd, token: capT, answer: capA})})
    setMsg(r.ok? 'Konto utworzone. Zaloguj się przez OIDC.' : 'Błąd przyjmowania zaproszenia')
  }
  return (<div className="card space-y">
    <h2 className="title">Dołącz z zaproszenia</h2>
    <p className="subtitle">{msg}</p>
    {ok && (<>
      <div className="row"><label className="label">Email</label><input className="input" value={email} onChange={e=>setEmail(e.target.value)} style={{width:360}}/></div>
      <div className="row"><Button onClick={async()=>{const r=await fetch('/api/core/captcha/challenge'); const j=await r.json(); setCapQ(j.question); setCapT(j.token)}}>CAPTCHA</Button><span className="label">{capQ}</span><Input value={capA} onChange={e=>setCapA(e.target.value)} style={{width:120}}/></div><div className="row"><label className="label">Hasło</label><input className="input" type="password" value={pwd} onChange={e=>setPwd(e.target.value)} style={{width:360}}/></div>
      <div className="row"><button className="btn btn-ok" onClick={accept}>Załóż konto</button></div>
    </>)}
  </div>)
}
