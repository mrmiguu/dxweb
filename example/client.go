package main

import "github.com/mrmiguu/dxweb"

func main() {
	for i, url := range []string{
		"assets/pics/mighty_no_09_cover_art_by_robduenas.jpg",
		"assets/pics/cougar_dragonsun.png",
		"assets/pics/trsipic1_lazur.jpg",
		"assets/pics/archmage_in_your_face.png",
		"assets/pics/acryl_bladerunner.png",
		"assets/pics/acryl_bobablast.png",
		"assets/pics/alex-bisleys_horsy_5.png",
	} {
		img := dxweb.NewImage(url, 600-i*90, 600-i*90)
		go func(i int) {
			img.Move(0, 0, 5000-i*250)
			img.Hide(true, 2500-i*125)
		}(i)
	}
}
