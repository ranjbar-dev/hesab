let accessToken = "";
const accessKey="admin_access_token", pendingKey="admin_pending_token";
export function getAccessToken(){if(!accessToken&&typeof window!=="undefined")accessToken=sessionStorage.getItem(accessKey)??"";return accessToken}
export function setAccessToken(token:string){accessToken=token;if(typeof window!=="undefined")sessionStorage.setItem(accessKey,token)}
export function clearSession(){accessToken="";if(typeof window!=="undefined"){sessionStorage.removeItem(accessKey);sessionStorage.removeItem(pendingKey)}}
export function setPending(token:string){sessionStorage.setItem(pendingKey,token)}
export function takePending(){const v=sessionStorage.getItem(pendingKey)??"";sessionStorage.removeItem(pendingKey);return v}
