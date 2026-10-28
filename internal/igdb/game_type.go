package igdb

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// GameType is the type of a game as defined by IGDB, such as main game, expansion or remake. See
// https://api-docs.igdb.com/#game
type GameType int

// The game types IGDB defines, with their IGDB IDs.
const (
	GameTypeMain                GameType = 0
	GameTypeDLCAddon            GameType = 1
	GameTypeExpansion           GameType = 2
	GameTypeBundle              GameType = 3
	GameTypeStandaloneExpansion GameType = 4
	GameTypeMod                 GameType = 5
	GameTypeEpisode             GameType = 6
	GameTypeSeason              GameType = 7
	GameTypeRemake              GameType = 8
	GameTypeRemaster            GameType = 9
	GameTypeExpandedGame        GameType = 10
	GameTypePort                GameType = 11
	GameTypeFork                GameType = 12
	GameTypePack                GameType = 13
	GameTypeUpdate              GameType = 14
)

// StandaloneGameTypes are the game types that can be searched and added to the library directly.
var StandaloneGameTypes = []GameType{
	GameTypeMain,
	GameTypeStandaloneExpansion,
	GameTypeRemake,
	GameTypeRemaster,
	GameTypeExpandedGame,
	GameTypePort,
	GameTypeFork,
}

// DLCGameTypes are the game types that count as DLC: they are managed from their parent game, never added directly.
var DLCGameTypes = []GameType{
	GameTypeDLCAddon,
	GameTypeExpansion,
	GameTypeEpisode,
}

// String returns the human readable string representation of the GameType.
func (gt GameType) String() string {
	switch gt {
	case GameTypeMain:
		return "Main"
	case GameTypeDLCAddon:
		return "DLC / Add-on"
	case GameTypeExpansion:
		return "Expansion"
	case GameTypeBundle:
		return "Bundle"
	case GameTypeStandaloneExpansion:
		return "Standalone Expansion"
	case GameTypeMod:
		return "Mod"
	case GameTypeEpisode:
		return "Episode"
	case GameTypeSeason:
		return "Season"
	case GameTypeRemake:
		return "Remake"
	case GameTypeRemaster:
		return "Remaster"
	case GameTypeExpandedGame:
		return "Expanded Game"
	case GameTypePort:
		return "Port"
	case GameTypeFork:
		return "Fork"
	case GameTypePack:
		return "Pack"
	case GameTypeUpdate:
		return "Update"
	default:
		return fmt.Sprintf("Unknown (%d)", int(gt))
	}
}

// Standalone reports whether gt is a standalone game type (StandaloneGameTypes): one that can be searched and added to
// the library directly.
func (gt GameType) Standalone() bool {
	return slices.Contains(StandaloneGameTypes, gt)
}

// DLC reports whether gt is a DLC type (DLCGameTypes): one that is managed from its parent game.
func (gt GameType) DLC() bool {
	return slices.Contains(DLCGameTypes, gt)
}

// gameTypeFilter returns types as a comma-separated list of IDs, for a game_type = (...) filter in an IGDB query.
func gameTypeFilter(types []GameType) string {
	str := make([]string, 0, len(types))
	for _, t := range types {
		str = append(str, strconv.Itoa(int(t)))
	}
	return strings.Join(str, ",")
}
