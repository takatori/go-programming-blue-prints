package meander

import "strings"

type j struct {
	Name      string
	PlaceType []string
}

var Journeys = []interface{}{
	&j{Name: "ロマンティック", PlaceType: []string{"park", "bar", "movie_theater", "restaurant", "florist", "taxi_stand"}},
	&j{Name: "ショッピング", PlaceType: []string{"department_store", "cafe", "clothing_store", "jewelry_store", "shoe_store"}},
	&j{Name: "ナイトライフ", PlaceType: []string{"bar", "casino", "food", "bar", "night_club", "bar", "bar", "hospital"}},
	&j{Name: "カルチャー", PlaceType: []string{"musemu", "cafe", "cemetery", "library", "art_gallery"}},
	&j{Name: "リラックス", PlaceType: []string{"hair_care", "beauty_salon", "cafe", "spa"}},
}

func (i *j) Public() interface{} {
	return map[string]interface{}{
		"name":    i.Name,
		"journey": strings.Join(i.PlaceType, "|"),
	}
}
