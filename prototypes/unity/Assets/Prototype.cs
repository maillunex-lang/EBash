using UnityEngine;
// Local functional prototype. No server, final assets or balanced economy.
public class Prototype : MonoBehaviour {
 Camera cam; GameObject city; Vector3 last; bool dragging, moved;
 string screen="", cityName="Уфа", message="Локальные демоданные";
 int money=100000,tick, selected, source=-1;
 string[] products={"Молоко","Бутылки","Крышки","Пастеризация","Розлив","Коктейль","Продажи","Продажи","Резерв"};
 [RuntimeInitializeOnLoadMethod(RuntimeInitializeLoadType.AfterSceneLoad)]
 static void Init(){new GameObject("Prototype").AddComponent<Prototype>();}
 void Start(){
  Screen.orientation=ScreenOrientation.LandscapeLeft; Application.targetFrameRate=30;
  city=new GameObject("City");
  for(int x=-3;x<=3;x++){
   Box(city.transform,new Vector3(x*12,0,0),new Vector3(2.4f,.12f,86),new Color(.28f,.34f,.39f));
   Box(city.transform,new Vector3(0,.01f,x*12),new Vector3(86,.12f,2.4f),new Color(.28f,.34f,.39f));
   for(int z=-3;z<=3;z++){
    int n=(x+3)*7+z+3;float h=2.5f+n%5*1.2f;
    var b=new GameObject("Building "+n);b.transform.parent=city.transform;b.transform.position=new Vector3(x*12+5,0,z*12+5);
    Box(b.transform,new Vector3(0,h/2,0),new Vector3(6,h,5),n%3==0?new Color(.41f,.67f,.71f):new Color(.85f,.83f,.75f));
    Box(b.transform,new Vector3(0,h+.2f,0),new Vector3(6.3f,.4f,5.3f),new Color(.2f,.27f,.34f));
    for(int f=0;f<(int)(h/1.3f);f++)for(int w=0;w<3;w++)Box(b.transform,new Vector3(-2+w*2,.8f+f*1.3f,2.52f),new Vector3(.8f,.7f,.06f),new Color(.78f,.91f,.93f));
   }
  }
  var l=new GameObject("Sun").AddComponent<Light>();l.type=LightType.Directional;l.intensity=1.25f;l.transform.rotation=Quaternion.Euler(55,-30,0);l.shadows=LightShadows.Soft;
  RenderSettings.ambientLight=new Color(.6f,.66f,.7f);
  cam=new GameObject("Camera").AddComponent<Camera>();cam.orthographic=true;cam.orthographicSize=31;cam.transform.position=new Vector3(55,58,55);cam.transform.LookAt(Vector3.zero);cam.backgroundColor=new Color(.72f,.82f,.87f);
 }
 void Box(Transform p,Vector3 pos,Vector3 size,Color color){var o=GameObject.CreatePrimitive(PrimitiveType.Cube);o.transform.SetParent(p,false);o.transform.localPosition=pos;o.transform.localScale=size;var m=new Material(Shader.Find("Standard"));m.color=color;o.GetComponent<Renderer>().sharedMaterial=m;}
 bool OverUI(Vector3 p){float sx=p.x*1280/Screen.width,sy=(Screen.height-p.y)*720/Screen.height;return sy<75||sy>660||sx<180||screen!="";}
 void Update(){
  if(Input.touchCount==2){var a=Input.GetTouch(0);var b=Input.GetTouch(1);float old=((a.position-a.deltaPosition)-(b.position-b.deltaPosition)).magnitude;cam.orthographicSize=Mathf.Clamp(cam.orthographicSize-(Vector2.Distance(a.position,b.position)-old)*.03f,11,55);return;}
  if(Input.GetMouseButtonDown(0)&&!OverUI(Input.mousePosition)){dragging=true;moved=false;last=Input.mousePosition;}
  if(dragging&&Input.GetMouseButton(0)){var d=Input.mousePosition-last;last=Input.mousePosition;if(d.magnitude>3)moved=true;cam.transform.position+=new Vector3(-d.x+d.y,0,d.x+d.y)*cam.orthographicSize/450;}
  if(Input.GetMouseButtonUp(0)&&dragging){dragging=false;if(!moved)screen="factory";}
  cam.orthographicSize=Mathf.Clamp(cam.orthographicSize-Input.mouseScrollDelta.y*1.5f,11,55);
 }
 bool Btn(float x,float y,float w,string t){return GUI.Button(new Rect(x,y,w,46),t);}
 void OnGUI(){
  GUI.matrix=Matrix4x4.TRS(Vector3.zero,Quaternion.identity,new Vector3(Screen.width/1280f,Screen.height/720f,1));GUI.skin.button.fontSize=18;GUI.skin.label.fontSize=20;GUI.skin.box.fontSize=20;
  GUI.Box(new Rect(0,0,1280,64),"");GUI.Label(new Rect(18,14,340,40),"EBASH / "+cityName+" • UNITY");GUI.Label(new Rect(360,14,340,40),money+" ₽ • Тик "+tick+" • ДЕМО");
  if(Btn(730,8,200,"Карта / Компания")){screen=screen=="company"?"":"company";city.SetActive(true);}
  if(Btn(940,8,140,"Тик +1")){tick++;money+=1250;}
  GUI.Label(new Rect(1090,14,180,40),Mathf.RoundToInt(1/Mathf.Max(Time.smoothDeltaTime,.001f))+" FPS");
  if(Btn(16,86,150,"Республика")){screen="republic";city.SetActive(false);}
  if(Btn(16,140,150,"Уфа")){screen="";cityName="Уфа";city.SetActive(true);}
  if(Btn(16,194,150,"Мой завод"))screen="factory";
  if(Btn(16,248,150,"+ Масштаб"))cam.orthographicSize=Mathf.Max(11,cam.orthographicSize-3);
  if(Btn(16,302,150,"− Масштаб"))cam.orthographicSize=Mathf.Min(55,cam.orthographicSize+3);
  if(Btn(16,356,150,"30 / 60 FPS"))Application.targetFrameRate=Application.targetFrameRate==30?60:30;
  GUI.Box(new Rect(0,666,1280,54),message+" • ПРОТОТИП / условные модели и показатели");
  if(screen=="")return;
  GUI.Box(new Rect(280,110,820,530),"");
  if(screen=="republic"){
   GUI.Label(new Rect(310,130,740,50),"РЕСПУБЛИКА БАШКОРТОСТАН / обзорная схема");
   string[] cities={"Уфа","Стерлитамак","Салават","Нефтекамск"};for(int i=0;i<4;i++)if(Btn(330,200+i*60,650,cities[i])){cityName=cities[i];city.SetActive(true);screen="";}
   GUI.Label(new Rect(310,465,740,70),"В прототипе города используют одну тестовую сцену.");
  }else if(screen=="company"){
   GUI.Label(new Rect(310,140,700,160),"КОМПАНИЯ / ОБЗОР\nБаланс: "+money+" ₽\nПредприятий: 1\nМолочный завод • загрузка 72%");
   if(Btn(330,350,620,"Открыть завод"))screen="factory";
  }else{
   GUI.Label(new Rect(310,130,700,50),"МОЛОЧНЫЙ ЗАВОД / ПРОИЗВОДСТВО");
   for(int i=0;i<9;i++)if(Btn(305+i%3*150,210+i/3*80,140,(i+1)+" "+products[i])){selected=i;if(source>=0){message="Связь: "+(source+1)+" → "+(i+1)+" (демо)";source=-1;}}
   GUI.Label(new Rect(775,210,300,130),"Ячейка "+(selected+1)+"\n"+products[selected]+"\nКачество: 38 / 100\nЗагрузка: 72%");
   if(Btn(775,355,290,"Сменить продукт"))products[selected]=products[selected]=="Коктейль"?"Розлив":"Коктейль";
   if(Btn(775,410,290,"Начать связь")){source=selected;message="Выберите получателя";}
   if(Btn(775,465,290,"Улучшить −5000 ₽")&&money>=5000)money-=5000;
   GUI.Label(new Rect(310,540,750,40),"Деморецепты. Серверная экономика не реализована.");
  }
  if(Btn(800,580,260,"Закрыть")){screen="";city.SetActive(true);}
 }
}
