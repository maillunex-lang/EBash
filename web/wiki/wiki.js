(()=>{
'use strict';
const $=id=>document.getElementById(id);
const make=(tag,cls,content)=>{const node=document.createElement(tag);if(cls)node.className=cls;if(content!==undefined)node.textContent=content;return node};
const statusName=s=>s==='approved'?'Утверждённый профиль':'Черновик DES-004';
const types={site:'Природный источник / объект',utility:'Коммунальный ресурс',external_good:'Внешний ресурс',produced_good:'Производимый товар'};
const qtyLabel=p=>p.quantity===null||p.quantity===undefined?'Количество не утверждено':new Intl.NumberFormat('ru-RU').format(p.quantity)+(p.unit?' '+p.unit:'');
const G={items:[],recipes:[],goods:[],familyNames:{},goodById:new Map(),recipeById:new Map(),makers:new Map(),users:new Map(),goodFamilies:new Map(),activeKind:'all',selected:null,ready:false};
const nice=id=>G.goodById.get(id)?.name||id.replaceAll('_',' ');
function addIndex(map,key,item){if(!map.has(key))map.set(key,[]);map.get(key).push(item)}
function show(type,id){G.userNavigated=true;const hash='#'+type+'/'+encodeURIComponent(id);if(window.location.hash===hash){G.selected={type,id};renderSelection();renderList();return}window.location.hash=hash}
function goRecipe(r){show('recipe',r.id)}
function goGood(id){show('good',id)}
function badge(status){return make('span','badge '+(status==='approved'?'ok':''),statusName(status))}
function nodeButton(id,part){const b=make('button','node');b.type='button';const name=make('strong','',nice(id));const small=make('small','',part?qtyLabel(part):'Товар / сырьё');b.append(name,small);b.title='Открыть товар «'+nice(id)+'»';b.addEventListener('click',()=>goGood(id));return b}
function recipeButton(recipe){const b=make('button','node '+(recipe.status==='draft'?'draftnode':''));b.type='button';b.append(make('strong','',recipe.name),make('small','',statusName(recipe.status)));b.title='Открыть рецепт';b.addEventListener('click',()=>goRecipe(recipe));return b}
function emptyNode(msg){return make('div','node',msg)}
function flow(leftTitle,leftItems,centerTitle,centerSub,rightTitle,rightItems,centerIsDraft){
 const shell=make('div','diagram');const grid=make('div','flow');
 const col=(title,items)=>{const c=make('div','flow-col');c.append(make('div','coltitle',title));for(const item of items)c.append(item);return c};
 const center=make('div','flow-col');center.append(make('div','coltitle','Выбранный объект'));
 const centerNode=make('div','node mainnode'+(centerIsDraft?' draftnode':''));centerNode.append(make('strong','',centerTitle),make('small','',centerSub));center.append(centerNode);
 grid.append(col(leftTitle,leftItems.length?leftItems:[emptyNode('Источник пока не описан')]),make('div','flow-arrow','→'),center,make('div','flow-arrow','→'),col(rightTitle,rightItems.length?rightItems:[emptyNode('Продолжение цепочки не описано')]));
 shell.append(grid);return shell;
}
function fact(key,value){const e=make('span','fact');e.append(document.createTextNode(key+' '),make('b','',String(value)));return e}
function linkList(title,arr,emptyMessage){const container=make('div','related-box');container.append(make('h3','',title));if(!arr.length){container.append(make('p','helper',emptyMessage));return container}
 for(const r of arr){const b=make('button','linkrow');b.type='button';b.append(make('span','',r.name),make('span','','→'));b.firstChild.style.color='inherit';b.addEventListener('click',()=>goRecipe(r));container.append(b)}return container}
function dedupe(arr){return Array.from(new Map(arr.map(r=>[r.id,r])).values())}
function addSection(parent,title,helper){parent.append(make('h3','section-title',title),make('p','helper',helper))}
function renderRecipe(r){
 const area=$('article');area.replaceChildren();const head=make('div','article-head');const left=make('div');
 left.append(make('div','group-label',G.familyNames[r.family]||r.family),make('h2','',r.name),make('div','subline',r.source_id+' · версия '+r.version));head.append(left,badge(r.status));area.append(head);
 area.append(make('div','hr'));addSection(area,'Схема производства','Нажми на любой компонент, чтобы посмотреть, как его получить или куда он используется. На телефоне схему можно листать по горизонтали.');
 const inputs=r.inputs.map(x=>nodeButton(x.good,x));const outputs=r.outputs.map(x=>nodeButton(x.good,x));
 area.append(flow('Что требуется',inputs,r.name,'Рецепт · '+statusName(r.status),'Что получится',outputs,r.status==='draft'));
 if(r.status==='approved'){
  const facts=make('div','facts');facts.append(fact('Минимальная партия:',r.minimum_batches),fact('Энергия:',r.energy_wh+' Вт·ч / партию'),fact('Работники:',r.workers),fact('Расходы:',new Intl.NumberFormat('ru-RU').format(r.cost_kopecks/100)+' ₽ / партию'),fact('Оборудование:',r.equipment));area.append(facts);
  area.append(make('div','notice','Параметры основаны на утверждённом игровом балансе. Это не промышленные нормативы. Выходы, не включённые в складской учёт, здесь не показаны как товар.'));
 }else area.append(make('div','notice','Это структурный черновик DES-004, а не исполняемый рецепт. Количества, оборудование, энергия и себестоимость пока не подтверждены и не подставляются автоматически.'));
 const upstream=dedupe(r.inputs.flatMap(p=>G.makers.get(p.good)||[])).filter(x=>x.id!==r.id);
 const downstream=dedupe(r.outputs.flatMap(p=>G.users.get(p.good)||[])).filter(x=>x.id!==r.id);
 area.append(make('div','hr'));addSection(area,'Соседние этапы цепочки','Проследи предыдущие производства или выбери следующий этап переработки.');const related=make('div','related');related.append(linkList('← Как получают сырьё',upstream,'Для входящего сырья нет описанного рецепта.'),linkList('Куда идёт продукт →',downstream,'Следующая переработка пока не описана.'));area.append(related)
}
function renderGood(g){
 const area=$('article');area.replaceChildren();const head=make('div','article-head');const left=make('div');
 left.append(make('div','group-label','Товар / ресурс'),make('h2','',g.name),make('div','subline',g.id+' · '+(types[g.kind]||g.kind)));head.append(left,make('span','badge ok','В каталоге DES-004'));area.append(head);
 const upstream=G.makers.get(g.id)||[],downstream=G.users.get(g.id)||[];
 area.append(make('div','hr'));addSection(area,'Связи в экономике','Слева рецепты получения, справа рецепты использования. Нажимай на карточки, чтобы перемещаться по производственной сети.');
 const max=7;const makeNodes=arr=>{const nodes=arr.slice(0,max).map(recipeButton);if(arr.length>max)nodes.push(emptyNode('Ещё '+(arr.length-max)+' рецептов в списке ниже'));return nodes};
 area.append(flow('Как получить',makeNodes(upstream),g.name,types[g.kind]||'Ресурс','Где используется',makeNodes(downstream),false));
 const facts=make('div','facts');facts.append(fact('Рецептов получения:',upstream.length),fact('Рецептов использования:',downstream.length));area.append(facts);
 area.append(make('div','hr'));addSection(area,'Карточки рецептов','Утверждённые игровые профили выделены зелёным, остальные являются черновиками.');const related=make('div','related');related.append(linkList('← Получение',upstream,'Нет рецепта получения в каталоге.'),linkList('Применение →',downstream,'Нет дальнейших рецептов в каталоге.'));area.append(related)
}
function renderSelection(){if(!G.ready)return;const s=G.selected;if(!s){$('article').replaceChildren(make('p','helper','Выбери товар или рецепт.'));return}
 const entry=s.type==='recipe'?G.recipeById.get(s.id):G.goodById.get(s.id);if(!entry){$('article').replaceChildren(make('p','helper','Такой записи нет в каталоге. Выбери другую.'));return}
 if(s.type==='recipe')renderRecipe(entry);else renderGood(entry);
 if(matchMedia('(max-width:680px)').matches&&(G.userNavigated||window.location.hash.startsWith('#')))$('article').scrollIntoView({behavior:matchMedia('(prefers-reduced-motion:reduce)').matches?'auto':'smooth',block:'start'});
}
function renderList(){if(!G.ready)return;const query=$('search').value.trim().toLocaleLowerCase('ru-RU');const fam=$('family').value;const filtered=G.items.filter(item=>{
 if(G.activeKind!=='all'&&item.type!==G.activeKind)return false;
 if(fam!=='all'&&(item.type==='recipe'?item.value.family!==fam:!G.goodFamilies.get(item.value.id)?.has(fam)))return false;
 if(!query)return true;
 const r=item.value;const text=item.type==='recipe'?[r.id,r.source_id,r.name,...r.inputs.map(x=>nice(x.good)),...r.outputs.map(x=>nice(x.good))].join(' '):[r.id,r.name].join(' ');
 return text.toLocaleLowerCase('ru-RU').includes(query)
 });$('count').textContent='Найдено: '+filtered.length+' из '+G.items.length;
 const holder=$('results');holder.replaceChildren();if(!filtered.length){holder.append(make('p','empty','Ничего не найдено. Попробуй другое название или выбери все отрасли.'));return}
 for(const item of filtered.slice(0,125)){
  const r=item.value;const active=G.selected?.type===item.type&&G.selected.id===r.id;
  const b=make('button','result'+(active?' selected':''));b.type='button';b.setAttribute('aria-current',String(active));
  b.append(make('span','glyph',item.type==='good'?'📦':r.status==='approved'?'⚙️':'📜'));
  const box=make('span','result-main');box.append(make('span','result-name',r.name),make('span','result-sub',item.type==='good'?(types[r.kind]||'Товар'):(G.familyNames[r.family]||r.family)));
  b.append(box);if(item.type==='recipe')b.append(make('span','tiny'+(r.status==='approved'?' ok':''),r.status==='approved'?'✓':'Черн.'));
  b.addEventListener('click',()=>{if(item.type==='recipe')goRecipe(r);else goGood(r.id)});holder.append(b)
 }
 if(filtered.length>125)holder.append(make('p','empty','Показаны первые 125 записей. Уточни поиск, чтобы найти остальные.'));
}
function syncHash(){if(!G.ready)return;let type,id;try{const match=window.location.hash.match(/^#(recipe|good)\/(.+)$/);if(match){type=match[1];id=decodeURIComponent(match[2])}}catch(e){/* invalid link */}
 if(!type){type='recipe';id='approved:GAME-BREAD-LOAF-V1'}
 G.selected={type,id};renderSelection();renderList()
}
function initialize(data){if(!Array.isArray(data.goods)||!Array.isArray(data.recipes))throw Error('Некорректный каталог');G.goods=data.goods;G.recipes=data.recipes;G.familyNames=data.families||{};
 for(const g of G.goods){G.goodById.set(g.id,g);G.goodFamilies.set(g.id,new Set())}
 for(const r of G.recipes){G.recipeById.set(r.id,r);for(const p of r.outputs){addIndex(G.makers,p.good,r);G.goodFamilies.get(p.good)?.add(r.family)}for(const p of r.inputs){addIndex(G.users,p.good,r);G.goodFamilies.get(p.good)?.add(r.family)}}
 G.items=[...G.recipes.filter(x=>x.status==='approved').map(value=>({type:'recipe',value})),...G.goods.slice().sort((a,b)=>a.name.localeCompare(b.name,'ru')).map(value=>({type:'good',value})),...G.recipes.filter(x=>x.status==='draft').sort((a,b)=>a.name.localeCompare(b.name,'ru')).map(value=>({type:'recipe',value}))];
 $('metrics').replaceChildren();const metric=(count,label,cls)=>{const box=make('span','metric '+cls);box.append(make('b','',String(count)),document.createTextNode(label));return box};
 $('metrics').append(metric(G.goods.length,'товаров и ресурсов',''),metric(G.recipes.filter(r=>r.status==='approved').length,'утверждённых профиля','approved'),metric(G.recipes.filter(r=>r.status==='draft').length,'черновых рецептов','draft'));
 for(const [id,title] of Object.entries(G.familyNames)){$('family').append(new Option(title,id))}
 G.ready=true;syncHash()
}
$('search').addEventListener('input',renderList);$('family').addEventListener('change',renderList);
document.querySelectorAll('[data-kind]').forEach(b=>b.addEventListener('click',()=>{G.activeKind=b.dataset.kind;for(const x of document.querySelectorAll('[data-kind]'))x.setAttribute('aria-pressed',String(x===b));renderList()}));
document.querySelectorAll('[data-good]').forEach(b=>b.addEventListener('click',()=>goGood(b.dataset.good)));
document.querySelectorAll('[data-recipe]').forEach(b=>b.addEventListener('click',()=>{const r=G.recipeById.get(b.dataset.recipe);if(r)goRecipe(r)}));
window.addEventListener('hashchange',syncHash);
fetch('./catalog.json',{cache:'no-cache'}).then(r=>{if(!r.ok)throw Error('HTTP '+r.status);return r.json()}).then(initialize).catch(err=>{$('results').replaceChildren(make('p','empty','Каталог временно недоступен.'));$('article').replaceChildren(make('p','helper','Не удалось загрузить данные. Обнови страницу позже.'));$('count').textContent='Ошибка загрузки';console.error('Wiki catalog:',err.message)})
})();
