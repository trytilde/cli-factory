package schema

type JSONSchema map[string]any

func PropertyNames(s JSONSchema) []string {
	props, ok := s["properties"].(map[string]any)
	if !ok {
		return nil
	}
	names := make([]string, 0, len(props))
	for name := range props {
		names = append(names, name)
	}
	return names
}
