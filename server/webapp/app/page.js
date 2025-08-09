async function getPosts(){ return [{id:1,title:'Witaj w Safe spac',body:'Dostęp tylko po 2FA.'}]; }
export default async function Home(){
  const posts = await getPosts();
  return (<div className="grid grid-2-1">
    <div className="">
      <h1 className="title">Posty</h1>
      {posts.map(p=> <div key={p.id}><h3>{p.title}</h3><p className="subtitle">{p.body}</p></div></Card>)}
      <a href="/app" className="">Przejdź do aplikacji</a>
    </div>
    <div className="">
      <h3 className="title">Logowanie</h3>
      <p className="subtitle">SSO via Authelia (OIDC).</p>
      <a href="/register" className="">Poproś o dostęp</a>
    </div>
  </div></Card>);
}
