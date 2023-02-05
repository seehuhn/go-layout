package layout

func (e *Engine) MakePage(height float64) []Box {
	if len(e.VList) == 0 {
		return nil
	}

	ext0 := e.VList[0].Extent()
	topSkip := e.TopSkip - ext0.Height
	if topSkip < 0 {
		topSkip = 0
	}

	total := &GlueBox{}
	for _, b := range e.VList {
		if glue, ok := b.(*GlueBox); ok {
			total = total.Add(glue)
		} else {
			ext := b.Extent()
			total.Length += ext.Height + ext.Depth
		}
	}
	panic("not implemented")
}
