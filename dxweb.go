package dxweb

import (
	"sync"

	"github.com/gopherjs/gopherjs/js"
	"github.com/mrmiguu/jsutil"
)

var (
	Width  = 800
	Height = 600

	start sync.Once

	phaser *js.Object
	game   *js.Object
	load   *js.Object

	objl sync.Mutex
	objs = map[string]chan<- *js.Object{}

	orderl sync.RWMutex
	orders []order
)

type order struct {
	key string
	ld  chan bool
}

func init() {
	style := js.Global.Get("document").Get("body").Get("style")
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
	mode := phaser.Get("ScaleManager").Get("SHOW_ALL")
	scale := game.Get("scale")
	scale.Set("scaleMode", mode)
	scale.Set("fullScreenScaleMode", mode)
	scale.Set("pageAlignHorizontally", true)
	scale.Set("pageAlignVertically", true)

	load = game.Get("load")
	load.Get("onFileComplete").Call("add", func(_, key *js.Object) {
		k := key.String()
		var obj *js.Object

		orderl.RLock()
		for i, ord := range orders {
			if ord.key != k {
				continue
			}
			if i > 0 {
				<-orders[i-1].ld
			}
			obj = game.Get("add").Call("image", game.Get("world").Get("centerX"), game.Get("world").Get("centerY"), k)
			ord.ld <- true
			break
		}
		orderl.RUnlock()

		if obj == nil {
			jsutil.Panic("object key not found")
		}

		objl.Lock()
		objc := objs[k]
		objl.Unlock()
		objc <- obj
	})
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

	orderl.Lock()
	orders = append(orders, order{url, make(chan bool, 1)})
	orderl.Unlock()

	objc := make(chan *js.Object)
	objl.Lock()
	objs[url] = objc
	objl.Unlock()
	load.Call("image", url, url)
	load.Call("start")

	c := make(chan Image)
	go func() {
		hit := make(chan bool, 1)
		img := Image{
			Hit: hit,
			key: url,
			js:  <-objc,
		}
		img.js.Set("alpha", 0)
		img.js.Get("anchor").Call("setTo", 0.5, 0.5)
		img.js.Get("events").Get("onInputDown").Call("add", func() { hit <- true })
		c <- img
	}()
	return c
}

func (i *Image) Pos() (int, int) {
	return i.js.Get("x").Int(), i.js.Get("y").Int()
}

func (i *Image) Size() (int, int) {
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
