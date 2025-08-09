'use client'
import {useEffect, useState} from 'react'
import {Card} from '@/components/ui/card'
export default function Health(){
  const [rows,setRows]=useState([])
  useEffect(()=>{ fetch('/api/core/admin/health',{headers:{'X-Role':'admin'}}).then(r=>r.json()).then(setRows).catch(()=>setRows([])) },[])
  return (<Card>
    <h2 className="text-lg mb-2">Status usług</h2>
    <ul>
      {rows.map((r,i)=><li key={i}><b>{r.name}</b>: {r.status}</li>)}
    </ul>
  </Card>)
}
