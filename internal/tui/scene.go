package tui

func (m Model) paintLogo(c *canvas, x, y int) {
	glyphs := []string{
		"1111010001111101000010000", // P
		"0111010001100011000101110", // O
		"1000110010111001001010001", // K
		"1111110000111101000011111", // E
		"0111110000100001000001111", // C
		"1111010001111101001010001", // R
		"1111100100001000010000100", // T
	}
	for i, g := range glyphs {
		for row := 0; row < 3; row++ {
			for col := 0; col < 5; col++ {
				upper := g[row*2*5+col] == '1'
				lower := false
				if row*2+1 < 5 {
					lower = g[(row*2+1)*5+col] == '1'
				}
				mark := " "
				if upper && lower {
					mark = "█"
				} else if upper {
					mark = "▀"
				} else if lower {
					mark = "▄"
				}
				c.put(x+i*6+col, y+row, mark, m.palette().accent)
			}
		}
	}
}
