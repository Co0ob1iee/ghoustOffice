import "../styles/globals.css"
export const metadata = { title: 'Safe spac' };
export default function RootLayout({children}) {
  return (<html lang="pl"><body><div className="container">{children}</div></body></html>);
}
