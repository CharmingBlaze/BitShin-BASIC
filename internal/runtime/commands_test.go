package runtime

import (
	"testing"

	"bitshinbasic/internal/value"
)

func TestCommandTablePillars(t *testing.T) {
	m := New(".").commandTable()
	need := []string{
		"graphics3d", "createcube", "createcamera", "renderworld", "flip",
		"graphics2d", "graphics", "rect", "oval", "drawimage", "createsprite",
		"loadsound", "playsound", "stopsound", "setsoundvolume", "setsoundpitch",
		"loadmusic", "playmusic", "emitsound",
		"createbodysphere", "createcircle2d", "setgravity",
		"raycast", "shapecast", "overlapsphere", "overlappoint", "applyimpulse", "createcharacter", "createcharactercontroller",
		"setcharactershape", "getcharactergroundstate", "getcharactergroundnormal",
		"getcharactercontact", "createpin2d", "createhinge", "createhingejoint", "createpointjoint",
		"createsliderjoint", "createspringjoint", "createballsocketjoint", "createjoint",
		"setbodyccd", "activatebody", "createbodycapsule", "setmass",
		"hostnet", "netrecv", "netsend",
		"jsonload", "jsonparse", "jsonget", "scenesave",
		"createnavmesh", "bakenavmesh", "createagent",
		"creategrid", "findpath", "pathlength",
		"guibegin", "guibutton", "guitext", "guislider",
		"ecsentity", "ecscomponent", "ecsquery", "ecsversion",
		"windowwidth", "setwindowsize", "keydown", "gamepadpresent",
		"createwindow", "freewindow", "setwindowcamera", "setrenderwindow",
		"windowclosed", "windowkeydown", "showwindow", "hidewindow",
		"enableshadows", "shadowmapsize", "shadowcascades",
		"setshadowbias", "setshadowpcf", "setshadowfilter", "setshadowpcss",
		"setshadowevsm", "setshadowmsm", "enableshadowcache", "enableshadowatlas",
		"enablecontactshadows", "enablescreenspaceshadows",
		"createdirectionallight", "setlightdirection", "setlightshadow",
		"entitycastshadow", "entityreceiveshadow", "setshadowdistance",
		"createemitter", "emittercolor", "emittervelocity", "emitterburst",
		"createemitter2d", "particle2drate",
		"createskybox", "loadskybox", "hideskybox", "setskycolor",
		"camerafollow", "entityspecular",
		"setweather", "setweatherintensity", "setfog",
		"setwind", "setclouds", "setatmosphere", "createatmosphere",
		"strikelightning", "setweatherwetness", "setfogheight", "setwindsway",
		"setweathertransition", "setweatherdryingspeed", "setsurfacewetness", "setcamerarain",
		"enablefog", "camerafogdensity", "createbox", "createcapsule",
		"jobsubmit", "jobwait", "jobwaitall", "createworldstream", "setstreamradius",
		"createinstancedmesh", "setinstancetransform", "instancecount",
		"createlightprobe", "setprobegrid",
		"createterrain", "createprocterrain", "terrainheight", "setterrainstreamradius",
		"createterraingl", "applyterrainsplat", "setterrainsplat", "setterrainoctaves",
		"setterrainfreq", "setterraindispfactor", "setterraingrasscoverage",
		"createproctexture", "createvolumetricclouds", "setcloudcoverage",
		"setskypreset", "setskygradient", "terrainslope",
		"setgeoorigin", "geoproject", "geounproject", "loadgeojson", "loadgeodem",
		"generateheightmap", "saveheightmap", "createterrainfromheightmap",
		"erodeheightmap", "hydraulicerodeheightmap", "filterheightmap",
		"createwater", "setwatercolor", "enablewaterreflection", "enablewaterrefraction", "setwaterdudv", "setwaterspeed", "setwaterwavespeed", "setwaterwind", "setwaterstyle", "setgerstner", "waterheight", "createbuoy",
		"setwatercaustics", "getwatercaustics", "setwaterssr", "getwaterssr", "setwaterambientsound", "getweatherintensity",
		"createcrowd", "crowdaddagent", "crowdupdate",
		"statsfps", "statschunks", "bodysleep", "physicsasync", "setshadowresolution",
		"createpbrmaterial", "setmaterialpbr", "enablepbr", "getmaterialpbr",
		"setmaterial", "setpbrmaterial", "setalbedo", "setbasecolor",
		"setalbedomap", "setbasecolormap", "getalbedor", "getbasecolorr",
		"getalbedog", "getbasecolorg", "getalbedob", "getbasecolorb",
		"setmetallic", "getmetallic", "setroughness", "getroughness",
		"setao", "getao", "setocclusion", "getocclusion", "setemissive",
		"getemissiver", "getemissiveg", "getemissiveb",
		"setnormalmap", "setmetallicroughnessmap", "setmetalroughmap",
		"setemissivemap", "setaomap", "setocclusionmap",
		"setenvmap", "setibl", "getibl", "setiblintensity", "getiblintensity",
		"getterrainheight", "getwaterheight", "getwaterspeed", "getjobcount", "getstatsfps",
		"createinstanced", "setwaterreflection", "addcrowdagent", "sleepbody",
		"getstreamchunkcount", "getentitycount", "setinstancecount",
		"createcomputeshader", "dispatchcompute", "createstoragebuffer",
		"createuniformbuffer", "setstoragebuffer", "setuniformbuffer",
		"compileshader", "creategeompoints", "enabletessellation",
		"enablegpuinstances", "setinstancedata", "glversion", "glhascompute",
		"noisefbm", "jobxform",
		"createpanel", "createbutton", "setonclick", "widgetvalue", "getwidgetvalue",
		"setgamepaddeadzone", "gamepaddeadzone", "getgamepaddeadzone",
		"getjobworkers", "getterrainstreamradius", "getterrainlod",
		"getwaterlevel", "getcrowdradius",
		"netpeercount", "netping", "netcall", "netregister",
		"createhost", "pollnetwork", "connecthost", "sendnet", "closehost", "getneteventtype", "getnetpeerip",
		"createnetworkhost", "createnetworkclient", "connectnetwork", "connect",
		"sendnetwork", "sendnetworkmessage", "disconnectnetwork", "closenetworkhost",
		"getnetworkeventtype", "getnetworkpeerid", "getnetworkdata", "getnetworkpeerip", "net_none",
		"creatememblock", "graphicswidth", "mousez",
		"enablepostfx", "setbloom", "setexposure", "setfxaa", "setcolorgrade",
		"createshader", "loadshader", "setshader", "setshaderuniform",
		"mouselook", "stopanim", "setanimblend", "setsky", "savescene",
		"createfreecamera", "updatefreelook",
		"createcharactercontroller", "setcharactershape", "getcharactergroundstate",
		"applytorque", "applyforceatposition", "applylocalimpulse", "setgravityscale",
		"createcarcontroller", "createvehicle", "setvehicleinput", "updatecar", "updatevehicle",
		"createplanecontroller", "updateplane", "createjetcontroller", "updatejet",
		"createspaceshipcontroller", "updatespaceship", "createboatcontroller", "updateboat",
		"createmotorcyclecontroller", "updatemotorcycle",
		"createhelicoptercontroller", "updatehelicopter",
		"createhovercraftcontroller", "updatehovercraft",
		"createsubmarinecontroller", "updatesubmarine",
		"createtankcontroller", "updatetank", "createtrackedcontroller", "updatetracked",
		"createdronecontroller", "updatedrone",
		"createwaterskicontroller", "updatewaterski",
		"createbodymesh", "createbodyheightfield", "createsensor", "setbodysensor",
		"offsetcenterofmass", "shapecast", "overlapsphere", "overlappoint", "optimizephysics",
		"createrope", "createropeanchored", "setropecolor", "setropemass",
		"setropedamping", "setropestrength", "setropevisible", "resetrope", "freerope",
		"ropelength", "ropetension", "ropesegments",
		"createcloth", "setclothwind", "setwaterflow", "setbuoyancyfactor",
		"createbodycylinder", "createbodyconvex", "grab", "grabpick", "dropgrab",
		"throw", "grabbedentity", "createprojectile", "createbeam", "placeatray", "attachtobone",
		"createbodycompound", "createhitbox", "explode", "followpath", "enablephysicsdebug",
		"createfixedjoint", "createconejoint", "createswingtwistjoint",
		"setcollisionlayer", "setlayercollides", "joint_fixed", "joint_cone", "joint_swingtwist",
	}
	for _, name := range need {
		if m[name] == nil {
			t.Errorf("missing command %s", name)
		}
	}
	if len(m) < 150 {
		t.Errorf("command table too small: %d (empty registries?)", len(m))
	}
}

func TestWeatherConstantCall(t *testing.T) {
	w := New(".")
	v, err := w.Call("weather_snow", nil)
	if err != nil {
		t.Fatal(err)
	}
	if v.String() != "snow" {
		t.Fatalf("WEATHER_SNOW → %q", v.String())
	}
}

func TestNetPollAndConstants(t *testing.T) {
	w := New(".")
	v, err := w.Call("net_connect", nil)
	if err != nil || v.Number() != 1 {
		t.Fatalf("NET_CONNECT %v %v", v, err)
	}
	none, err := w.Call("net_none", nil)
	if err != nil || none.Number() != 0 {
		t.Fatalf("NET_NONE %v %v", none, err)
	}
	disc, err := w.Call("net_disconnect", nil)
	if err != nil || disc.Number() != 2 {
		t.Fatalf("NET_DISCONNECT %v %v", disc, err)
	}
	recv, err := w.Call("net_receive", nil)
	if err != nil || recv.Number() != 3 {
		t.Fatalf("NET_RECEIVE %v %v", recv, err)
	}
	ev, err := w.Call("pollnetwork", nil)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Kind != value.KindStruct {
		t.Fatalf("PollNetwork kind %v", ev.Kind)
	}
	typ, ok := ev.Field("type")
	if !ok || typ.Number() != 0 {
		t.Fatalf("empty event type %v %v", typ, ok)
	}
	if _, ok := ev.Field("peerid"); !ok {
		t.Fatal("missing PeerID")
	}
	if _, ok := ev.Field("peerip"); !ok {
		t.Fatal("missing PeerIP")
	}
	if _, ok := ev.Field("data"); !ok {
		t.Fatal("missing Data")
	}
	hinge, err := w.Call("createhinge", nil)
	if err != nil || hinge.Number() != 0 {
		t.Fatalf("CreateHinge must return 0, got %v %v", hinge, err)
	}
}
