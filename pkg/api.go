package pkg

type AddPostFormValues struct {
	Post string
}

func (values AddPostFormValues) Validate() map[string]string {
	errors := make(map[string]string)
	if len(values.Post) < 3 {
		errors["Post"] = "too short"
	}

	if len(values.Post) > 10 {
		errors["Post"] = "too long"
	}

	if values.Post == "bad" {
		errors["Post"] = "bad not allowed"
	}

	return errors
}
