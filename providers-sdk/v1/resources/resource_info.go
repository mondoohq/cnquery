package resources

// HasEmptyInit is true if the init call has non-optional arguments
// or if the resource has mandatory fields. In either case it
// cannot be called without arguments.
func (r *ResourceInfo) HasEmptyInit() bool {
	if r.Init != nil {
		for i := range r.Init.Args {
			if !r.Init.Args[i].Optional {
				return false
			}
		}
	}

	for _, v := range r.Fields {
		if v.IsMandatory {
			return false
		}
	}

	return true
}
