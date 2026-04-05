package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/stockyard-dev/stockyard-collection/internal/store"
)

type Server struct {
	store  *store.Store
	port   int
	limits Limits
	mux    *http.ServeMux
}

func New(s *store.Store, port int, limits Limits) *Server {
	srv := &Server{store: s, port: port, limits: limits, mux: http.NewServeMux()}
	srv.routes()
	return srv
}

func (s *Server) Start() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), s.mux)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/categories", s.hListCategories)
	s.mux.HandleFunc("POST /api/categories", s.hCreateCategory)
	s.mux.HandleFunc("DELETE /api/categories/{id}", s.hDeleteCategory)
	s.mux.HandleFunc("GET /api/items", s.hListItems)
	s.mux.HandleFunc("POST /api/items", s.hCreateItem)
	s.mux.HandleFunc("GET /api/items/{id}", s.hGetItem)
	s.mux.HandleFunc("PUT /api/items/{id}", s.hUpdateItem)
	s.mux.HandleFunc("DELETE /api/items/{id}", s.hDeleteItem)
	s.mux.HandleFunc("GET /api/stats", s.hStats)
	s.mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		j(w, 200, map[string]string{"status": "ok", "product": "collection"})
	})
	s.mux.HandleFunc("GET /api/limits", func(w http.ResponseWriter, r *http.Request) { j(w, 200, s.limits) })
	s.mux.HandleFunc("GET /ui", s.hUI)
	s.mux.HandleFunc("GET /ui/", s.hUI)
	s.mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" { http.Redirect(w, r, "/ui", 307); return }
		http.NotFound(w, r)
	})
}

func (s *Server) hListCategories(w http.ResponseWriter, r *http.Request) {
	cats, _ := s.store.ListCategories()
	if cats == nil { cats = []store.Category{} }
	j(w, 200, cats)
}

func (s *Server) hCreateCategory(w http.ResponseWriter, r *http.Request) {
	if LimitReached(s.limits.MaxCategories, s.store.CategoryCount()) {
		j(w, 402, map[string]string{"error": fmt.Sprintf("Free tier limit: %d categories", s.limits.MaxCategories)})
		return
	}
	var body struct{ Name string `json:"name"`; Color string `json:"color"` }
	json.NewDecoder(r.Body).Decode(&body)
	if body.Name == "" { j(w, 400, map[string]string{"error": "name required"}); return }
	id, err := s.store.CreateCategory(body.Name, body.Color)
	if err != nil { j(w, 500, map[string]string{"error": err.Error()}); return }
	j(w, 201, map[string]any{"id": id, "name": body.Name})
}

func (s *Server) hDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	s.store.DeleteCategory(id)
	j(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) hListItems(w http.ResponseWriter, r *http.Request) {
	catID, _ := strconv.ParseInt(r.URL.Query().Get("category_id"), 10, 64)
	search := r.URL.Query().Get("q")
	limit := 200
	if l := r.URL.Query().Get("limit"); l != "" { if n, _ := strconv.Atoi(l); n > 0 { limit = n } }
	items, _ := s.store.ListItems(catID, search, limit)
	if items == nil { items = []store.Item{} }
	j(w, 200, items)
}

func (s *Server) hCreateItem(w http.ResponseWriter, r *http.Request) {
	if LimitReached(s.limits.MaxItems, s.store.ItemCount()) {
		j(w, 402, map[string]string{"error": fmt.Sprintf("Free tier limit: %d items", s.limits.MaxItems)})
		return
	}
	var item store.Item
	json.NewDecoder(r.Body).Decode(&item)
	if item.Name == "" { j(w, 400, map[string]string{"error": "name required"}); return }
	id, err := s.store.CreateItem(item)
	if err != nil { j(w, 500, map[string]string{"error": err.Error()}); return }
	item.ID = id
	j(w, 201, item)
}

func (s *Server) hGetItem(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	item, err := s.store.GetItem(id)
	if err != nil { j(w, 404, map[string]string{"error": "not found"}); return }
	j(w, 200, item)
}

func (s *Server) hUpdateItem(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	var item store.Item
	json.NewDecoder(r.Body).Decode(&item)
	item.ID = id
	s.store.UpdateItem(item)
	j(w, 200, item)
}

func (s *Server) hDeleteItem(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
	s.store.DeleteItem(id)
	j(w, 200, map[string]string{"status": "deleted"})
}

func (s *Server) hStats(w http.ResponseWriter, r *http.Request) { j(w, 200, s.store.GetStats()) }

func j(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func (s *Server) hUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(uiHTML))
}

const uiHTML = `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>Collection — Stockyard</title>
<style>*{margin:0;padding:0;box-sizing:border-box}:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#c45d2c;--rust-light:#e8753a;--leather:#a0845c;--cream:#f0e6d3;--cream-dim:#bfb5a3;--gold:#d4a843;--font:'JetBrains Mono',monospace}body{background:var(--bg);color:var(--cream);font-family:var(--font);font-size:0.82rem}.header{padding:1rem 1.5rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center}.header h1{font-size:0.9rem;color:var(--rust-light);letter-spacing:2px;text-transform:uppercase}.main{padding:1.5rem}.stat-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(140px,1fr));gap:1rem;margin-bottom:1.5rem}.stat{background:var(--bg2);border:1px solid var(--bg3);padding:1rem;text-align:center}.stat-val{font-size:1.5rem;color:var(--cream)}.stat-label{font-size:0.65rem;color:var(--leather);text-transform:uppercase;letter-spacing:1px}input,select,textarea{background:var(--bg2);border:1px solid var(--bg3);color:var(--cream);padding:0.5rem;font-family:var(--font);font-size:0.8rem;width:100%}input:focus{border-color:var(--rust);outline:none}button{background:var(--rust);color:var(--cream);border:none;padding:0.5rem 1rem;cursor:pointer;font-family:var(--font);font-size:0.75rem}button:hover{background:var(--rust-light)}.item-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(250px,1fr));gap:1rem}.item-card{background:var(--bg2);border:1px solid var(--bg3);padding:1rem;cursor:pointer;transition:border-color 0.2s}.item-card:hover{border-color:var(--rust)}.item-name{color:var(--cream);font-size:0.85rem;margin-bottom:0.3rem}.item-meta{color:var(--leather);font-size:0.7rem}.form-row{margin-bottom:0.6rem}.form-row label{display:block;color:var(--leather);font-size:0.65rem;text-transform:uppercase;letter-spacing:1px;margin-bottom:0.2rem}.empty{text-align:center;padding:3rem;color:var(--leather)}.search{margin-bottom:1.5rem;display:flex;gap:0.5rem}.cat-pills{display:flex;gap:0.5rem;flex-wrap:wrap;margin-bottom:1rem}.cat-pill{padding:0.3rem 0.8rem;background:var(--bg2);border:1px solid var(--bg3);cursor:pointer;font-size:0.7rem;color:var(--cream-dim)}.cat-pill:hover,.cat-pill.active{border-color:var(--rust);color:var(--rust-light)}</style></head>
<body><div class="header"><h1>Collection</h1><span style="color:var(--leather);font-size:0.7rem">Stockyard</span></div>
<div class="main"><div class="stat-grid" id="stats"></div>
<div class="search"><input id="searchInput" placeholder="Search items..." onkeyup="loadItems()"><button onclick="showAdd()">+ Add Item</button><button onclick="showAddCat()" style="background:var(--bg3)">+ Category</button></div>
<div class="cat-pills" id="catPills"></div>
<div id="addForm" style="display:none;background:var(--bg2);border:1px solid var(--bg3);padding:1rem;margin-bottom:1rem">
<div class="form-row"><label>Name *</label><input id="iName"></div>
<div style="display:grid;grid-template-columns:1fr 1fr;gap:0.5rem">
<div class="form-row"><label>Category</label><select id="iCat"></select></div>
<div class="form-row"><label>Value ($)</label><input id="iValue" type="number" step="0.01" value="0"></div>
</div>
<div class="form-row"><label>Location</label><input id="iLoc"></div>
<div class="form-row"><label>Notes</label><textarea id="iNotes" rows="2"></textarea></div>
<button onclick="createItem()">Save</button> <button onclick="hideAdd()" style="background:var(--bg3)">Cancel</button>
</div>
<div id="addCatForm" style="display:none;background:var(--bg2);border:1px solid var(--bg3);padding:1rem;margin-bottom:1rem">
<div class="form-row"><label>Category Name</label><input id="cName"></div>
<button onclick="createCat()">Save</button> <button onclick="hideAddCat()" style="background:var(--bg3)">Cancel</button>
</div>
<div class="item-grid" id="items"></div>
</div>
<script>
let activeCat=0;
async function load(){const s=await(await fetch('/api/stats')).json();document.getElementById('stats').innerHTML='<div class="stat"><div class="stat-val">'+s.total_items+'</div><div class="stat-label">Items</div></div><div class="stat"><div class="stat-val">'+s.total_categories+'</div><div class="stat-label">Categories</div></div><div class="stat"><div class="stat-val">$'+(s.total_value_cents/100).toFixed(2)+'</div><div class="stat-label">Total Value</div></div>';
const cats=await(await fetch('/api/categories')).json();let pills='<div class="cat-pill'+(activeCat===0?' active':'')+'" onclick="filterCat(0)">All</div>';let opts='<option value="0">None</option>';cats.forEach(c=>{pills+='<div class="cat-pill'+(activeCat===c.id?' active':'')+'" onclick="filterCat('+c.id+')">'+c.name+' ('+c.count+')</div>';opts+='<option value="'+c.id+'">'+c.name+'</option>'});document.getElementById('catPills').innerHTML=pills;document.getElementById('iCat').innerHTML=opts;loadItems()}
async function loadItems(){const q=document.getElementById('searchInput').value;let url='/api/items?limit=200';if(activeCat>0)url+='&category_id='+activeCat;if(q)url+='&q='+encodeURIComponent(q);const items=await(await fetch(url)).json();if(items.length===0){document.getElementById('items').innerHTML='<div class="empty">No items yet</div>';return}let h='';items.forEach(i=>{h+='<div class="item-card"><div class="item-name">'+i.name+'</div><div class="item-meta">'+(i.category||'No category')+(i.value_cents>0?' · $'+(i.value_cents/100).toFixed(2):'')+(i.location?' · '+i.location:'')+'</div>'+(i.notes?'<div class="item-meta" style="margin-top:0.3rem;color:var(--cream-dim)">'+i.notes+'</div>':'')+'</div>'});document.getElementById('items').innerHTML=h}
function filterCat(id){activeCat=id;load()}
function showAdd(){document.getElementById('addForm').style.display='block'}
function hideAdd(){document.getElementById('addForm').style.display='none'}
function showAddCat(){document.getElementById('addCatForm').style.display='block'}
function hideAddCat(){document.getElementById('addCatForm').style.display='none'}
async function createItem(){const b={name:document.getElementById('iName').value,category_id:parseInt(document.getElementById('iCat').value)||0,value_cents:Math.round(parseFloat(document.getElementById('iValue').value||'0')*100),location:document.getElementById('iLoc').value,notes:document.getElementById('iNotes').value};if(!b.name){alert('Name required');return}await fetch('/api/items',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b)});hideAdd();document.getElementById('iName').value='';load()}
async function createCat(){const n=document.getElementById('cName').value;if(!n)return;await fetch('/api/categories',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name:n})});hideAddCat();document.getElementById('cName').value='';load()}
load();</script></body></html>`
