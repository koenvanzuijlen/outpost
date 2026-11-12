package server

// ratingStarFill is how much of one rating star is filled. The value is also the modifier of the star's CSS class.
type ratingStarFill string

// The fills of a rating star.
const (
	ratingStarEmpty ratingStarFill = "empty"
	ratingStarHalf  ratingStarFill = "half"
	ratingStarFull  ratingStarFill = "full"
)

// ratingStarSide is the half of a rating star that one rating input option shows. The value is also the modifier of
// the option's CSS class.
type ratingStarSide string

// The halves of a rating star.
const (
	ratingStarLeft  ratingStarSide = "left"
	ratingStarRight ratingStarSide = "right"
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

// ratingInputOption is one half star of the rating input: the rating it sets, its accessible label and its side.
type ratingInputOption struct {
	Value int64
	Label string
	Side  ratingStarSide
}

// ratingInputOptions returns the options of the rating input for the ratings 1 to 2*ratingStarCount, in ascending
// order. An odd rating is the left half of a star and an even rating is the right half.
func ratingInputOptions() []ratingInputOption {
	options := make([]ratingInputOption, 0, ratingStarCount*2)
	for i := int64(1); i <= ratingStarCount*2; i++ {
		side := ratingStarLeft
		if i%2 == 0 {
			side = ratingStarRight
		}
		options = append(options, ratingInputOption{
			Value: i,
			Label: ratingLabel(i),
			Side:  side,
		})
	}
	return options
}
