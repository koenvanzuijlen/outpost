package igdb

import "fmt"

// GameType as defined by the IGDB API. See https://api-docs.igdb.com/#game
type GameType int

// GameType as defined by the IGDB API. See https://api-docs.igdb.com/#game
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
