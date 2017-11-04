package dxweb

import (
	"image/png"
	"net/http"
	"strings"
	"sync"

	"github.com/gopherjs/gopherjs/js"
	"github.com/mrmiguu/jsutil"
)

var (
	Width  = 800
	Height = 600

	start sync.Once

	body   *js.Object
	phaser *js.Object
	game   *js.Object
	load   *js.Object
	add    *js.Object

	centerX, centerY int

	orderl sync.RWMutex
	orders []order
)

type order struct {
	key  string
	keyc chan string
	ld   chan bool
}

func init() {
	style := js.Global.Get("document").Get("body").Get("style")

	document := js.Global.Get("document")
	body = document.Get("body")
	body.Get("style").Set("visibility", "hidden")

	meta := document.Call("createElement", "meta")
	meta.Set("name", "viewport")
	meta.Set("content", "width=device-width, initial-scale=1, maximum-scale=1, user-scalable=0")
	body.Call("appendChild", meta)

	style.Set("background", "#000000")
	style.Set("margin", 0)
	<-jsutil.Load("assets/js/phaser.min.js")
}

func run() {
	f, c := jsutil.C()
	phaser = js.Global.Get("Phaser")
	game = phaser.Get("Game").New(Width, Height, nil, nil, js.M{"create": f})
	<-c

	game.Get("canvas").Set("oncontextmenu", func(e *js.Object) { e.Call("preventDefault") })

	scale := game.Get("scale")
	window := js.Global.Get("window")
	if iW, iH := window.Get("innerWidth").Float(), window.Get("innerHeight").Float(); Height > Width {
		newW := (float64(Width) / float64(Height)) * iH
		scale.Call("setMinMax", newW, iH, newW, iH)
	} else {
		newH := (float64(Height) / float64(Width)) * iW
		scale.Call("setMinMax", iW, newH, iW, newH)
	}

	mode := phaser.Get("ScaleManager").Get("SHOW_ALL")
	scale.Set("scaleMode", mode)
	scale.Set("pageAlignHorizontally", true)
	scale.Set("pageAlignVertically", true)

	js.Global.Call("setTimeout", func() {
		body.Get("style").Set("visibility", "visible")
	}, 200)

	centerX, centerY = game.Get("world").Get("centerX").Int(), game.Get("world").Get("centerY").Int()

	load = game.Get("load")
	load.Get("onFileComplete").Call("add", func(_, key *js.Object) {
		go func() {
			k := key.String()

			orderl.RLock()
			for i, ord := range orders {
				if ord.key != k {
					continue
				}
				if i > 0 {
					<-orders[i-1].ld
				}
				ord.keyc <- k
				break
			}
			orderl.RUnlock()
		}()
	})

	add = game.Get("add")
}

func tween(obj *js.Object, to js.M, ms ...int) {
	millis := getMS(ms...)
	if millis == 0 {
		for k, v := range to {
			obj.Set(k, v)
		}
		return
	}
	move := add.Call("tween", obj)
	move.Call("to", to, millis)
	move.Set("frameBased", true)
	f, c := jsutil.C()
	move.Get("onComplete").Call("add", f)
	move.Call("start")
	<-c
}

func getMS(ms ...int) int {
	if len(ms) > 1 {
		jsutil.Panic("too many arguments")
	}
	if len(ms) > 0 {
		if ms[0] < 0 {
			jsutil.Panic("negative ms")
		}
		return ms[0]
	}
	return 0
}

type Image struct {
	Hit <-chan bool
	key string
	js  *js.Object
}

func LoadImage(url string) <-chan Image {
	start.Do(run)

	ord := order{url, make(chan string, 1), make(chan bool, 1)}
	orderl.Lock()
	orders = append(orders, ord)
	orderl.Unlock()

	load.Call("image", url, url)
	load.Call("start")

	imgc := make(chan Image)
	go func() {
		obj := add.Call("image", game.Get("world").Get("centerX"), game.Get("world").Get("centerY"), <-ord.keyc)
		ord.ld <- true

		obj.Set("alpha", 0)
		obj.Get("anchor").Call("setTo", 0.5, 0.5)

		hit := make(chan bool)
		obj.Get("events").Get("onInputDown").Call("add", jsutil.F(func(_ ...*js.Object) { hit <- true }))

		imgc <- Image{
			Hit: hit,
			key: url,
			js:  obj,
		}
	}()
	return imgc
}

func (i Image) Pos() (int, int) {
	return i.js.Get("x").Int(), i.js.Get("y").Int()
}

func (i Image) Size() (int, int) {
	return i.js.Get("width").Int(), i.js.Get("height").Int()
}

func (i *Image) Move(x, y int, ms ...int) {
	tween(i.js, js.M{"x": x, "y": y}, ms...)
}

func (i *Image) Resize(width, height int, ms ...int) {
	tween(i.js, js.M{"width": width, "height": height}, ms...)
}

func (i *Image) Show(b bool, ms ...int) {
	a := 1
	if !b {
		a = 0
		i.js.Set("inputEnabled", false)
	}
	tween(i.js, js.M{"alpha": a}, ms...)
	if b {
		i.js.Set("inputEnabled", true)
	}
}

type Sprite struct {
	Hit    <-chan bool
	key    string
	frames int
	anims  []<-chan []*js.Object
	js     *js.Object
}

func LoadSprite(url string, frames, states int) <-chan Sprite {
	start.Do(run)

	ord := order{url, make(chan string, 1), make(chan bool, 1)}
	orderl.Lock()
	orders = append(orders, ord)
	orderl.Unlock()

	resp, err := http.Get(url)
	if err != nil {
		jsutil.Panic(err)
	}
	img, err := png.Decode(resp.Body)
	resp.Body.Close()
	if err != nil {
		jsutil.Panic(err)
	}
	size := img.Bounds().Size()
	load.Call("spritesheet", url, url, size.X/frames, size.Y/states, frames*states)
	load.Call("start")

	sprc := make(chan Sprite)
	go func() {
		obj := add.Call("sprite", game.Get("world").Get("centerX"), game.Get("world").Get("centerY"), <-ord.keyc)
		ord.ld <- true

		anims := make([]<-chan []*js.Object, states)
		for i := 0; i < states; i++ {
			s := make(js.S, frames)
			for j := 0; j < frames; j++ {
				s[j] = (i * frames) + j
			}
			anim := obj.Get("animations").Call("add", i, s)
			f, c := jsutil.C()
			anim.Get("onComplete").Call("add", f)
			anims[i] = c
		}
		obj.Get("animations").Set("frame", frames-1)
		obj.Set("alpha", 0)
		obj.Get("anchor").Call("setTo", 0.5, 0.5)

		hit := make(chan bool)
		obj.Get("events").Get("onInputDown").Call("add", jsutil.F(func(_ ...*js.Object) { hit <- true }))

		sprc <- Sprite{
			Hit:    hit,
			key:    url,
			frames: frames,
			anims:  anims,
			js:     obj,
		}
	}()
	return sprc
}

func (s Sprite) Pos() (int, int) {
	return s.js.Get("x").Int(), s.js.Get("y").Int()
}

func (s Sprite) Size() (int, int) {
	return s.js.Get("width").Int(), s.js.Get("height").Int()
}

func (s *Sprite) Move(x, y int, ms ...int) {
	tween(s.js, js.M{"x": x, "y": y}, ms...)
}

func (s *Sprite) Rotate(θ float64, ms ...int) {
	tween(s.js, js.M{"angle": θ}, ms...)
}

func (s *Sprite) Resize(width, height int, ms ...int) {
	tween(s.js, js.M{"width": width, "height": height}, ms...)
}

func (s *Sprite) Show(b bool, ms ...int) {
	a := 1
	if !b {
		a = 0
		s.js.Set("inputEnabled", false)
	}
	tween(s.js, js.M{"alpha": a}, ms...)
	if b {
		s.js.Set("inputEnabled", true)
	}
}

func (s Sprite) Play(state int, ms ...int) {
	millis := getMS(ms...)
	fps := 60
	if millis > 0 {
		fps = s.frames * 1000 / millis
	}
	s.js.Get("animations").Call("play", state, fps)
	<-s.anims[state]
}

type Sound struct {
	key string
	js  *js.Object
}

func LoadSound(url string) <-chan Sound {
	start.Do(run)

	ord := order{url, make(chan string, 1), make(chan bool, 1)}
	orderl.Lock()
	orders = append(orders, ord)
	orderl.Unlock()
	load.Call("audio", url, url)
	load.Call("start")

	sfxc := make(chan Sound)
	go func() {
		obj := add.Call("audio", <-ord.keyc)
		ord.ld <- true

		sfxc <- Sound{
			key: url,
			js:  obj,
		}
	}()
	return sfxc
}

func (s Sound) Play() {
	s.js.Call("play")
}

type Text struct {
	Hit <-chan bool
	js  *js.Object
}

func NewText(lines ...string) Text {
	obj := add.Call("text", centerX, centerY, strings.Join(lines, "\n"))
	obj.Set("alpha", 0)
	obj.Set("align", "center")
	obj.Get("anchor").Call("set", 0.5)
	obj.Set("font", "Arial")
	obj.Set("fontWeight", "normal")
	obj.Set("fontSize", "12")
	obj.Set("fill", "#ffffff")
	obj.Call("setShadow", 0, -1, "rgba(0,0,0,1)", 1)

	hit := make(chan bool)
	obj.Get("events").Get("onInputDown").Call("add", jsutil.F(func(_ ...*js.Object) { hit <- true }))

	return Text{
		Hit: hit,
		js:  obj,
	}
}

func (t Text) Pos() (int, int) {
	return t.js.Get("x").Int(), t.js.Get("y").Int()
}

func (t *Text) Move(x, y int, ms ...int) {
	tween(t.js, js.M{"x": x, "y": y}, ms...)
}

func (t Text) Size() (int, int) {
	return t.js.Get("width").Int(), t.js.Get("height").Int()
}

func (t Text) Get() string {
	return t.js.Get("text").String()
}

func (t *Text) Set(lines ...string) {
	t.js.Set("text", strings.Join(lines, "\n"))
}

func (t *Text) Recolor(hex string) {
	t.js.Set("fill", hex)
}

func (t *Text) Resize(size int, ms ...int) {
	tween(t.js, js.M{"fontSize": size}, ms...)
}

func (t *Text) Show(b bool, ms ...int) {
	a := 1
	if !b {
		a = 0
		t.js.Set("inputEnabled", false)
	}
	tween(t.js, js.M{"alpha": a}, ms...)
	if b {
		t.js.Set("inputEnabled", true)
	}
}
