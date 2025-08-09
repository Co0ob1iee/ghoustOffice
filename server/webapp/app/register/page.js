'use client'\nimport {Button} from '@/components/ui/button'\nimport {Input} from '@/components/ui/input'\nimport {Card} from '@/components/ui/card'
import {useState} from 'react'
export default function Register(){
  const [email,setEmail]=useState('')
  const [note,setNote]=useState('')
  const [msg,setMsg] = useState(''); const [capQ,setCapQ]=useState(''); const [capT,setCapT]=useState(''); const [capA,setCapA]=useState('')=useState('')
  const submit= async ()=>{
    const r = await fetch('/api/core/registration/request',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({email,note, CaptchaToken: capT, CaptchaAnswer: capA})})
    setMsg(r.ok? 'Wniosek złożony. Poczekaj na akceptację.' : 'Błąd')
  }
  return (<div className=" space-y">
    <h2 className="title">Rejestracja (na zatwierdzenie)</h2>
    <p className="subtitle">Podaj email i krótko uzasadnij dostęp. Admin zatwierdzi wniosek.</p>
    <div className="row"><Button className="" onClick={async()=>{const r=await fetch('/api/core/captcha/challenge'); const j=await r.json(); setCapQ(j.question); setCapT(j.token)}}>CAPTCHA</Button><span className="label">{capQ}</span><Input className="" value={capA} onChange={e=>setCapA(e.target.value)} style={{width:120}}/></div><div className="row">
      <label className="label">Email</label>
      <Input className="input" value={email} onChange={e=>setEmail(e.target.value)} style={{width:320}}/>
    </div>
    <div className="row">
      <label className="label">Uzasadnienie</label>
      <Input className="input" value={note} onChange={e=>setNote(e.target.value)} style={{width:460}}/>
    </div>
    <div className="row">
      <Button className=" btn-ok" onClick={submit}>Wyślij wniosek</Button>
    </div>
    {msg && <div className="alert">{msg}</div>}
  </div></Card>)
}
