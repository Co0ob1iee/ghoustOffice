'use client'
import {useEffect, useState} from 'react'
export default function Admin(){
  const [users,setUsers]=useState([])
  useEffect(()=>{
    const token = window.localStorage.getItem('access_token') || ''
    fetch('/api/core/admin/users', { headers: {'X-Role':'admin','Authorization':'Bearer '+token} })
      .then(r=>r.json()).then(setUsers).catch(()=>setUsers([]))
  },[])
  return (<div style={{background:'#0f1724',border:'1px solid #1e2a3d',borderRadius:18,padding:22}}>
    <h2>Panel administracyjny</h2>
    <p>Użytkownicy (placeholder):</p>
    <pre>{JSON.stringify(users,null,2)}</pre>
  <p><a className="btn" href="/admin/invite">Zaproszenia</a> <a className="btn" style={{marginLeft:8}} href="/admin/pending">Wnioski</a></p><p><a className="btn" href="/admin/health">Health</a></p></div>)
}
