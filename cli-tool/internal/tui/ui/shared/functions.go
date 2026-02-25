package shared

func SizeBuffer() (width, height int) {
	x, y := DocStyle.GetFrameSize()
	return WindowSize.Width - x, WindowSize.Height - y
}
