package main

// varieties is the whole catalogue of forms, assembled from the tables kept
// beside the species they belong to.
var varieties = map[string][]Variety{}

func init() {
	for _, table := range []map[string][]Variety{
		flowerVarieties, bulbVarieties, herbVarieties,
		edibleVarieties, treeVarieties, wildVarieties,
	} {
		for id, vs := range table {
			varieties[id] = vs
		}
	}
}
