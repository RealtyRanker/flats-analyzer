package geo

// sadovoeRing is the Sadovoye Koltso boundary, given as (lat, lon) vertices
// in order; the ring is implicitly closed (the last point connects back to
// the first).
var sadovoeRing = newRing([]point{
	{55.748405, 37.582478},
	{55.758942, 37.584622},
	{55.764362, 37.588656},
	{55.769805, 37.596423},
	{55.772491, 37.604534},
	{55.773410, 37.611830},
	{55.773652, 37.617495},
	{55.773603, 37.625391},
	{55.772781, 37.632472},
	{55.772007, 37.637922},
	{55.768741, 37.649295},
	{55.763999, 37.656333},
	{55.761918, 37.657449},
	{55.758409, 37.657792},
	{55.756231, 37.657578},
	{55.752407, 37.656076},
	{55.745194, 37.655303},
	{55.742434, 37.654445},
	{55.737447, 37.648565},
	{55.731611, 37.638738},
	{55.730691, 37.633888},
	{55.729819, 37.622859},
	{55.729844, 37.612044},
	{55.735461, 37.594320},
	{55.738682, 37.586124},
	{55.746719, 37.582433},
	{55.748220, 37.582433},
})

// InsideSadovoeRing reports whether (lat, lon) falls inside the Sadovoye
// Koltso polygon.
func InsideSadovoeRing(lat, lon float64) bool {
	return sadovoeRing.inside(lat, lon)
}

// DistanceToSadovoeRingKM returns the shortest distance in kilometers from
// (lat, lon) to the Sadovoye Koltso boundary.
func DistanceToSadovoeRingKM(lat, lon float64) float64 {
	return sadovoeRing.distanceKM(lat, lon)
}
