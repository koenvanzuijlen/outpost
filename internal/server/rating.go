package server

// ratingStarFill is how much of one rating star is filled. The value is also the modifier of the star's CSS class.
type ratingStarFill string

// The fills of a rating star.
const (
	ratingStarEmpty ratingStarFill = "empty"
	ratingStarHalf  ratingStarFill = "half"
	ratingStarFull  ratingStarFill = "full"
)

// ratingStarCount is the number of stars that show a rating. Each star is 2 rating points.
const ratingStarCount = 5

// ratingStarFills returns the fill of each of the ratingStarCount stars for a rating from 1 to 10. Each star takes 2
// rating points: 2 fill the star and 1 fills half of it. A rating below 1 gives empty stars and a rating above 10 gives
// full stars.
func ratingStarFills(rating int64) []ratingStarFill {
	fills := make([]ratingStarFill, 0, ratingStarCount)
	for i := range int64(ratingStarCount) {
		rest := rating - 2*i
		switch {
		case rest >= 2:
			fills = append(fills, ratingStarFull)
		case rest == 1:
			fills = append(fills, ratingStarHalf)
		default:
			fills = append(fills, ratingStarEmpty)
		}
	}
	return fills
}
