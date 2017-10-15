package dxweb

import (
	"sync"

	"github.com/gopherjs/gopherjs/js"
	"github.com/mrmiguu/jsutil"
)

var (
	Width    = 800
	Height   = 600
	HitSound = ""

	start sync.Once

	phaser *js.Object
	game   *js.Object
	load   *js.Object

	orderl sync.RWMutex
	orders []order

	ldhit  sync.Once
	hitsfx Sound
)

type order struct {
	key  string
	keyc chan string
	ld   chan bool
}

func init() {
	style := js.Global.Get("document").Get("body").Get("style")

	document := js.Global.Get("document")
	meta := document.Call("createElement", "meta")
	meta.Set("name", "viewport")
	meta.Set("content", "width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=0")
	document.Get("body").Call("appendChild", meta)

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
	innerWidth, innerHeight := window.Get("innerWidth").Float(), window.Get("innerHeight").Float()
	if Height > Width {
		newWidth := (float64(Width) / float64(Height)) * innerHeight
		scale.Call("setMinMax", newWidth, innerHeight, newWidth, innerHeight)
	} else {
		newHeight := (float64(Height) / float64(Width)) * innerWidth
		scale.Call("setMinMax", innerWidth, newHeight, innerWidth, newHeight)
	}

	mode := phaser.Get("ScaleManager").Get("SHOW_ALL")
	scale.Set("scaleMode", mode)
	scale.Set("pageAlignHorizontally", true)
	scale.Set("pageAlignVertically", true)
	scale.Call("refresh")

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
}

func loadHit() {
	if len(HitSound) == 0 {
		return
	}
	hitsfx = <-LoadSound(HitSound)
}

func tween(obj *js.Object, to js.M, ms ...int) {
	move := game.Get("add").Call("tween", obj)
	move.Call("to", to, getMS(ms...))
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
		if ms[0] < 1 {
			jsutil.Panic("negative or zero ms")
		}
		return ms[0]
	}
	return 1
}

type Image struct {
	Hit <-chan bool
	key string
	js  *js.Object
}

func LoadImage(url string) <-chan Image {
	start.Do(run)
	ldhit.Do(loadHit)

	ord := order{url, make(chan string, 1), make(chan bool, 1)}
	orderl.Lock()
	orders = append(orders, ord)
	orderl.Unlock()
	load.Call("image", url, url)
	load.Call("start")

	imgc := make(chan Image)
	go func() {
		obj := game.Get("add").Call("image", game.Get("world").Get("centerX"), game.Get("world").Get("centerY"), <-ord.keyc)
		ord.ld <- true

		obj.Set("alpha", 0)
		obj.Get("anchor").Call("setTo", 0.5, 0.5)

		hit := make(chan bool)
		obj.Get("events").Get("onInputDown").Call("add", jsutil.F(func() {
			if hitsfx.js != nil {
				hitsfx.Play()
			}
			hit <- true
		}))

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
		obj := game.Get("add").Call("audio", <-ord.keyc)
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
