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
		img.Show(true, 2500)
		go func(img dxweb.Image, i int) {
			width, height := img.Size()
			go img.Resize(-width, -height, 5000-i*250)
			img.Move(width/2, height/2, 5000-i*250)
			img.Show(false, 2500-i*125)
		}(img, i)
	}
}
