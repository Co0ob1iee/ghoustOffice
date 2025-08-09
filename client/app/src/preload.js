const { contextBridge, ipcRenderer } = require('electron')

contextBridge.exposeInMainWorld('electronAPI', {
  oidcLogin: (issuer) => ipcRenderer.invoke('oidc-login', issuer),
  issueProfile: (issuer, mode, scope) => ipcRenderer.invoke('issue-profile', issuer, mode, scope),
})
