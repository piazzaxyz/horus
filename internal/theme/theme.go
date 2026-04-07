package theme

// Theme holds all color values for a visual theme.
type Theme struct {
	Name       string
	Background string
	Foreground string
	Primary    string
	Secondary  string
	Accent     string
	Success    string
	Warning    string
	Error      string
	Info       string
	Border     string
	Highlight  string
	Muted      string
	Comment    string
	Critical   string
	High       string
	Medium     string
	Low        string
}

var themes = map[string]Theme{
	"Tokyo Night": {
		Name:       "Tokyo Night",
		Background: "#1a1b2e",
		Foreground: "#c0caf5",
		Primary:    "#7aa2f7",
		Secondary:  "#bb9af7",
		Accent:     "#2ac3de",
		Success:    "#9ece6a",
		Warning:    "#e0af68",
		Error:      "#f7768e",
		Info:       "#7dcfff",
		Border:     "#3b4261",
		Highlight:  "#283457",
		Muted:      "#565f89",
		Comment:    "#444b6a",
		Critical:   "#f7768e",
		High:       "#ff9e64",
		Medium:     "#e0af68",
		Low:        "#9ece6a",
	},
	"Catppuccin Mocha": {
		Name:       "Catppuccin Mocha",
		Background: "#1e1e2e",
		Foreground: "#cdd6f4",
		Primary:    "#89b4fa",
		Secondary:  "#cba6f7",
		Accent:     "#89dceb",
		Success:    "#a6e3a1",
		Warning:    "#f9e2af",
		Error:      "#f38ba8",
		Info:       "#74c7ec",
		Border:     "#45475a",
		Highlight:  "#313244",
		Muted:      "#6c7086",
		Comment:    "#585b70",
		Critical:   "#f38ba8",
		High:       "#fab387",
		Medium:     "#f9e2af",
		Low:        "#a6e3a1",
	},
	"Dracula": {
		Name:       "Dracula",
		Background: "#282a36",
		Foreground: "#f8f8f2",
		Primary:    "#bd93f9",
		Secondary:  "#ff79c6",
		Accent:     "#8be9fd",
		Success:    "#50fa7b",
		Warning:    "#f1fa8c",
		Error:      "#ff5555",
		Info:       "#6272a4",
		Border:     "#44475a",
		Highlight:  "#44475a",
		Muted:      "#6272a4",
		Comment:    "#6272a4",
		Critical:   "#ff5555",
		High:       "#ffb86c",
		Medium:     "#f1fa8c",
		Low:        "#50fa7b",
	},
	"Nord": {
		Name:       "Nord",
		Background: "#2e3440",
		Foreground: "#d8dee9",
		Primary:    "#88c0d0",
		Secondary:  "#81a1c1",
		Accent:     "#5e81ac",
		Success:    "#a3be8c",
		Warning:    "#ebcb8b",
		Error:      "#bf616a",
		Info:       "#b48ead",
		Border:     "#3b4252",
		Highlight:  "#434c5e",
		Muted:      "#4c566a",
		Comment:    "#4c566a",
		Critical:   "#bf616a",
		High:       "#d08770",
		Medium:     "#ebcb8b",
		Low:        "#a3be8c",
	},
	"Gruvbox Dark": {
		Name:       "Gruvbox Dark",
		Background: "#282828",
		Foreground: "#ebdbb2",
		Primary:    "#83a598",
		Secondary:  "#b16286",
		Accent:     "#689d6a",
		Success:    "#b8bb26",
		Warning:    "#fabd2f",
		Error:      "#cc241d",
		Info:       "#458588",
		Border:     "#504945",
		Highlight:  "#3c3836",
		Muted:      "#928374",
		Comment:    "#7c6f64",
		Critical:   "#cc241d",
		High:       "#d65d0e",
		Medium:     "#d79921",
		Low:        "#98971a",
	},
	"One Dark": {
		Name:       "One Dark",
		Background: "#282c34",
		Foreground: "#abb2bf",
		Primary:    "#61afef",
		Secondary:  "#c678dd",
		Accent:     "#56b6c2",
		Success:    "#98c379",
		Warning:    "#e5c07b",
		Error:      "#e06c75",
		Info:       "#61afef",
		Border:     "#3e4451",
		Highlight:  "#2c313c",
		Muted:      "#5c6370",
		Comment:    "#5c6370",
		Critical:   "#e06c75",
		High:       "#d19a66",
		Medium:     "#e5c07b",
		Low:        "#98c379",
	},
	"Everforest": {
		Name:       "Everforest",
		Background: "#2d353b",
		Foreground: "#d3c6aa",
		Primary:    "#7fbbb3",
		Secondary:  "#d699b6",
		Accent:     "#83c092",
		Success:    "#a7c080",
		Warning:    "#dbbc7f",
		Error:      "#e67e80",
		Info:       "#7fbbb3",
		Border:     "#475258",
		Highlight:  "#374145",
		Muted:      "#859289",
		Comment:    "#5c6a72",
		Critical:   "#e67e80",
		High:       "#e69875",
		Medium:     "#dbbc7f",
		Low:        "#a7c080",
	},
}

// ThemeNames returns all available theme names in order.
var ThemeNames = []string{
	"Tokyo Night",
	"Catppuccin Mocha",
	"Dracula",
	"Nord",
	"Gruvbox Dark",
	"One Dark",
	"Everforest",
}

// Get returns the theme by name, defaulting to Tokyo Night.
func Get(name string) Theme {
	if t, ok := themes[name]; ok {
		return t
	}
	return themes["Tokyo Night"]
}

// All returns all themes as a slice.
func All() []Theme {
	result := make([]Theme, 0, len(ThemeNames))
	for _, name := range ThemeNames {
		result = append(result, themes[name])
	}
	return result
}

// Next returns the name of the theme after the given one.
func Next(current string) string {
	for i, name := range ThemeNames {
		if name == current {
			return ThemeNames[(i+1)%len(ThemeNames)]
		}
	}
	return ThemeNames[0]
}
