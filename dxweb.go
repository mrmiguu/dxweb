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

func pos(o *js.Object) (int, int) {
	return o.Get("x").Int(), o.Get("y").Int()
}

func size(o *js.Object) (int, int) {
	return o.Get("width").Int(), o.Get("height").Int()
}

func bringToTop(o *js.Object) {
	o.Call("bringToTop")
}

func move(o *js.Object, x, y int, ms ...int) {
	tween(o, js.M{"x": x, "y": y}, ms...)
}

func resize(o *js.Object, width, height int, ms ...int) {
	tween(o, js.M{"width": width, "height": height}, ms...)
}

func show(o *js.Object, b bool, disabled bool, ms ...int) {
	a := 1
	if !b {
		a = 0
		disable(o, true)
	}
	tween(o, js.M{"alpha": a}, ms...)
	if b && !disabled {
		disable(o, false)
	}
}

func rotate(o *js.Object, θ float64, ms ...int) {
	tween(o, js.M{"angle": θ}, ms...)
}

func disable(o *js.Object, b bool) {
	o.Set("inputEnabled", !b)
}

type Rect interface {
	Pos() (int, int)
	Size() (int, int)
}

func Top(r Rect, ms ...int) (int, int, int) {
	x, _ := r.Pos()
	_, height := r.Size()
	return x, height / 2, getMS(ms...)
}

func Left(r Rect, ms ...int) (int, int, int) {
	_, y := r.Pos()
	width, _ := r.Size()
	return width / 2, y, getMS(ms...)
}

func Bottom(r Rect, ms ...int) (int, int, int) {
	x, _ := r.Pos()
	_, height := r.Size()
	return x, Height - (height / 2), getMS(ms...)
}

func Right(r Rect, ms ...int) (int, int, int) {
	_, y := r.Pos()
	width, _ := r.Size()
	return Width - (width / 2), y, getMS(ms...)
}

type Image struct {
	Hit <-chan bool

	disabled bool
	key      string
	js       *js.Object
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
		obj.Get("events").Get("onInputDown").Call("add", func() {
			go func() {
				println("hit...")
				hit <- true
				println("hit!")
			}()
		})

		imgc <- Image{
			Hit: hit,
			key: url,
			js:  obj,
		}
	}()
	return imgc
}

func (i *Image) LoadImage(url string) <-chan Image {
	start.Do(run)

	ord := order{url, make(chan string, 1), make(chan bool, 1)}
	orderl.Lock()
	orders = append(orders, ord)
	orderl.Unlock()

	load.Call("image", url, url)
	load.Call("start")

	imgc := make(chan Image)
	go func() {
		obj := game.Get("make").Call("image", 0, 0, <-ord.keyc)
		ord.ld <- true

		obj.Set("alpha", 0)
		obj.Get("anchor").Call("setTo", 0.5, 0.5)

		hit := make(chan bool)
		obj.Get("events").Get("onInputDown").Call("add", func() {
			go func() {
				println("hit...")
				hit <- true
				println("hit!")
			}()
		})

		imgc <- Image{
			Hit: hit,
			key: url,
			js:  i.js.Call("addChild", obj),
		}
	}()
	return imgc
}

func (i Image) Pos() (int, int) {
	return pos(i.js)
}

func (i Image) Size() (int, int) {
	return size(i.js)
}

func (i *Image) Move(x, y int, ms ...int) {
	move(i.js, x, y, ms...)
}

func (i *Image) Rotate(θ float64, ms ...int) {
	rotate(i.js, θ, ms...)
}

func (i *Image) Resize(width, height int, ms ...int) {
	resize(i.js, width, height, ms...)
}

func (i *Image) Show(b bool, ms ...int) {
	show(i.js, b, i.disabled, ms...)
}

func (i *Image) Disable(b bool) {
	i.disabled = b
	disable(i.js, b)
	if !b {
		i.js.Get("input").Set("pixelPerfectAlpha", 1)
		i.js.Get("input").Set("pixelPerfectClick", true)
	}
}

func (i *Image) BringToTop() {
	bringToTop(i.js)
}

type Sprite struct {
	Hit <-chan bool

	disabled bool
	key      string
	frames   int
	anims    []<-chan bool
	js       *js.Object
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

		anims := make([]<-chan bool, states)
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
		obj.Get("events").Get("onInputDown").Call("add", func() {
			go func() {
				println("hit...")
				hit <- true
				println("hit!")
			}()
		})

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
	return pos(s.js)
}

func (s Sprite) Size() (int, int) {
	return size(s.js)
}

func (s *Sprite) Move(x, y int, ms ...int) {
	move(s.js, x, y, ms...)
}

func (s *Sprite) Rotate(θ float64, ms ...int) {
	rotate(s.js, θ, ms...)
}

func (s *Sprite) Resize(width, height int, ms ...int) {
	resize(s.js, width, height, ms...)
}

func (s *Sprite) Show(b bool, ms ...int) {
	show(s.js, b, s.disabled, ms...)
}

func (s *Sprite) Disable(b bool) {
	disable(s.js, b)
	if !b {
		s.js.Get("input").Set("pixelPerfectAlpha", 1)
		s.js.Get("input").Set("pixelPerfectClick", true)
	}
}

func (s *Sprite) Play(state int, ms ...int) {
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

func (s Sound) Loop() {
	s.js.Call("loopFull")
}

type Text struct {
	Hit <-chan bool

	disabled bool
	js       *js.Object
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
	obj.Get("events").Get("onInputDown").Call("add", func() {
		go func() {
			println("hit...")
			hit <- true
			println("hit!")
		}()
	})

	return Text{
		Hit: hit,
		js:  obj,
	}
}

func (t Text) Pos() (int, int) {
	return pos(t.js)
}

func (t Text) Size() (int, int) {
	return size(t.js)
}

func (t Text) Get() string {
	return t.js.Get("text").String()
}

func (t *Text) Set(lines ...string) {
	t.js.Set("text", strings.Join(lines, "\n"))
}

func (t *Text) Move(x, y int, ms ...int) {
	tween(t.js, js.M{"x": x, "y": y}, ms...)
}

func (t *Text) Recolor(hex string) {
	t.js.Set("fill", hex)
}

func (t *Text) Resize(size int, ms ...int) {
	tween(t.js, js.M{"fontSize": size}, ms...)
}

func (t *Text) Show(b bool, ms ...int) {
	show(t.js, b, t.disabled, ms...)
}

func (t *Text) Disable(b bool) {
	disable(t.js, b)
}
