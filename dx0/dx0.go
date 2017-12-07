package dx0

import (
	"math"

	"github.com/gopherjs/gopherjs/js"
)

type Surface struct {
	parent *Surface
	js     *js.Object
}

func (s Surface) Offset(xratio, y float64) {
	math.Abs
}
