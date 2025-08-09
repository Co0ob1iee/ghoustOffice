'use client'
import {useEffect, useMemo, useState} from 'react'
import {Card} from '@/components/ui/card'
import {Input} from '@/components/ui/input'
import {Button} from '@/components/ui/button'

export default function Audit(){
  const [rows,setRows]=useState([])
  const [q,setQ]=useState('')
  useEffect(()=>{
    fetch('/api/core/admin/audit',{headers:{'X-Role':'admin'}})
      .then(r=>r.json()).then(setRows).catch(()=>setRows([]))
  },[])
  const filtered = useMemo(()=> rows.filter(r => !q || JSON.stringify(r).toLowerCase().includes(q.toLowerCase())), [rows,q])
  return (
    <Card>
      <h2 className="text-lg mb-2">Audit log</h2>
      <div className="flex items-center gap-2 mb-3">
        <Input placeholder="Szukaj..." value={q} onChange={e=>setQ(e.target.value)} style={{maxWidth:320}}/>
        <Button onClick={()=>setQ('')}>Wyczyść</Button>
      </div>
      <div style={{overflowX:'auto'}}>
        <table style={{width:'100%', borderCollapse:'collapse'}}>
          <thead>
            <tr><th style={{textAlign:'left'}}>Czas (UTC)</th><th>Akcja</th><th>Aktor</th><th>Szczegóły</th></tr>
          </thead>
          <tbody>
            {filtered.map((r,i)=>(
              <tr key={i}>
                <td>{r.Ts}</td><td>{r.Action}</td><td>{r.Actor}</td><td>{r.Details}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  )
}
