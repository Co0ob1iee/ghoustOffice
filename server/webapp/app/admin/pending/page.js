'use client'\nimport {Button} from '@/components/ui/button'\nimport {Input} from '@/components/ui/input'\nimport {Card} from '@/components/ui/card'
import {useEffect, useState} from 'react'
export default function Pending(){
  const [items,setItems]=useState([])
  const load = ()=>{
    fetch('/api/core/admin/pending',{headers:{'X-Role':'admin'}}).then(r=>r.json()).then(setItems).catch(()=>setItems([]))
  }
  useEffect(load,[])
  const decide = async (email,decision)=>{
    await fetch('/api/core/admin/pending/decision',{method:'POST',headers:{'X-Role':'admin','Content-Type':'application/json'},body:JSON.stringify({email,decision})})
    load()
  }
  return (<div style={{background:'#0f1724',border:'1px solid #1e2a3d',borderRadius:18,padding:22}}>
    <h2>Wnioski rejestracyjne</h2>
    <ul>
      {items.map((it,i)=> <li key={i}>{it.email} — {it.status} {it.status==='pending' && (<>
        <Button onClick={()=>decide(it.email,'approve')}>Zatwierdź</Button>
        <Button onClick={()=>decide(it.email,'reject')} style={{marginLeft:8}}>Odrzuć</Button>
      </>)}</li>)}
    </ul>
  </div></Card>)
}
