package web

import "net/http"

func AssetHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/style.css" {
		w.Header().Set("Content-Type", "text/css")
		w.Write([]byte(css))
		return
	}
	if r.URL.Path == "/app.js" {
		w.Header().Set("Content-Type", "application/javascript")
		w.Write([]byte(js))
		return
	}
	http.NotFound(w, r)
}

const css = `body{font-family:system-ui,sans-serif;max-width:1000px;margin:0 auto;padding:32px;color:#17202a;background:#f5f7f9}section{background:#fff;border:1px solid #d9e0e6;border-radius:8px;padding:20px;margin:18px 0}input,button{padding:10px;margin:4px;border:1px solid #b8c2cc;border-radius:4px}button{background:#155eef;color:white;cursor:pointer}pre{white-space:pre-wrap;background:#f0f3f6;padding:12px}.muted{color:#5b6570}`
const js = `let current=null;const $=id=>document.getElementById(id);async function call(path,method='GET',body){let r=await fetch(path,{method,headers:{'Content-Type':'application/json'},body:body?JSON.stringify(body):undefined});let j=await r.json();if(!r.ok)throw Error(j.error||j.message);return j}function show(c){current=c;$('panel').hidden=false;$('status').textContent=(c.statusLabel||c.status)+' · '+c.performanceName;$('detail').textContent=JSON.stringify(c,null,2)}$('create').onsubmit=async e=>{e.preventDefault();let o=Object.fromEntries(new FormData(e.target));try{show(await call('/api/cases','POST',o))}catch(x){alert(x.message)}};const cid=()=>current.caseId;$('revision').onclick=async()=>{try{show(await call('/api/cases/'+cid()+'/revisions','POST',{expectedVersion:current.expectedVersion,idempotencyKey:crypto.randomUUID(),note:'提交方案',by:current.ownerName,points:[{pointCode:'A',stageZone:current.stageZones,componentName:'主吊点',ratedLoadKg:1000,actualLoadKg:600,clearanceMm:800,positionXmm:0,positionYmm:0,cueStart:1,cueEnd:10}]}))}catch(x){alert(x.message)}};$('assess').onclick=async()=>{try{let r=await call('/api/cases/'+cid()+'/assess','POST');$('result').textContent=JSON.stringify(r,null,2);show(await call('/api/cases/'+cid()))}catch(x){alert(x.message)}};async function rev(stage){try{show(await call('/api/cases/'+cid()+'/reviews','POST',{expectedVersion:current.expectedVersion,stage,outcome:'Approve',reviewer:stage==='Safety'?'安全复核员':'技术负责人',comment:'同意',revisionDigest:current.revisions.at(-1).contentDigest}))}catch(x){alert(x.message)}};$('review').onclick=()=>rev('Safety');$('technical').onclick=()=>rev('Technical');$('freeze').onclick=async()=>{try{let p=await call('/api/cases/'+cid()+'/freeze','POST',{expectedVersion:current.expectedVersion,by:'技术负责人'});$('result').textContent=JSON.stringify(p,null,2);show(await call('/api/cases/'+cid()))}catch(x){alert(x.message)}};$('verify').onclick=async()=>{try{$('result').textContent=JSON.stringify(await call('/api/permits/'+$('permit').value),null,2)}catch(x){alert(x.message)}};`
