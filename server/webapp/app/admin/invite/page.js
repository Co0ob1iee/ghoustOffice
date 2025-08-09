'use client'\nimport {Button} from '@/components/ui/button'\nimport {Input} from '@/components/ui/input'\nimport {Card} from '@/components/ui/card'
import {useState} from 'react'
export default function InviteAdmin(){
  const [email,setEmail]=useState(''); const [hrs,setHrs]=useState(48); const [link,setLink]=useState('')
  const make = async ()=>{
    const r = await fetch('/api/core/admin/invite',{method:'POST',headers:{'X-Role':'admin','Content-Type':'application/json'},body:JSON.stringify({email, validHours: parseInt(hrs)})})
    const j = await r.json().catch(()=>({}))
    if(j.token){
      const base = window.location.origin
      setLink(base+'/invite/'+j.token)
    } else setLink('Błąd')
  }
  return (<div className=" space-y">
    <h2 className="title">Zaproszenia</h2>
    <p className="subtitle">Wygeneruj link ważny X godzin. Opcjonalnie „przypnij” do konkretnego emaila.</p>
    <div className="row"><label className="label">Email (opcjonalny)</label><Input className="input" value={email} onChange={e=>setEmail(e.target.value)} style={{width:300}}/></div>
    <div className="row"><label className="label">Ważność [h]</label><Input className="input" type="number" value={hrs} onChange={e=>setHrs(e.target.value)} style={{width:120}}/></div>
    <div className="row"><Button className=" btn-ok" onClick={make}>Generuj</Button></div>
    {link && <div className="alert">Link: <a href={link}>{link}</a></div>}
  </div></Card>)
}
