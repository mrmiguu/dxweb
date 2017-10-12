package dxweb

import (
	"sync"

	"github.com/gopherjs/gopherjs/js"
	"github.com/mrmiguu/jsutil"
)

var (
	start sync.Once
	game  *js.Object

	imagel sync.Mutex
	images = map[string]imageLoader{}
)

func init() {
	<-jsutil.Load("assets/js/phaser.min.js")
}

func run() {
	f, c := jsutil.C()
	game = js.Global.Get("Phaser").Get("Game").New(800, 600, nil, nil, js.M{"create": f})
	<-c
	game.Get("load").Get("onFileComplete").Call("add", func(_, key *js.Object) {
		obj := game.Get("add").Call("image", game.Get("world").Get("centerX"), game.Get("world").Get("centerY"), key)
		obj.Set("alpha", 0)
		obj.Get("anchor").Call("setTo", 0.5, 0.5)

		imagel.Lock()
		img := images[key.String()]
		imagel.Unlock()
		wh := <-img.wh
		obj.Set("width", wh[0])
		obj.Set("height", wh[1])
		img.js <- obj

		fade := game.Get("add").Call("tween", obj).Call("to", js.M{"alpha": 1}, 2500)
		fade.Set("frameBased", true)
		fade.Call("start")
	})
}

type imageLoader struct {
	wh <-chan [2]int
	js chan<- *js.Object
}

type Image struct {
	key           string
	width, height int
	js            *js.Object
}

func NewImage(url string, width, height int) *Image {
	start.Do(run)

	game.Get("load").Call("image", url, url)
	game.Get("load").Call("start")

	wh := make(chan [2]int, 1)
	wh <- [2]int{width, height}
	obj := make(chan *js.Object, 1)
	imagel.Lock()
	images[url] = imageLoader{wh, obj}
	imagel.Unlock()

	return &Image{
		key:    url,
		width:  width,
		height: height,
		js:     <-obj,
	}
}

func (i *Image) Move(x, y int, ms ...int) {
	if len(ms) > 1 {
		panic("too many arguments")
	}
	millis := 1
	if len(ms) > 0 {
		if ms[0] < 1 {
			panic("negative or zero ms")
		}
		millis = ms[0]
	}
	move := game.Get("add").Call("tween", i.js).Call("to", js.M{"x": x, "y": y}, millis)
	move.Set("frameBased", true)
	move.Call("start")
}
