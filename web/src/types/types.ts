export let CDRAGON = 'https://raw.communitydragon.org/latest'

export type ChampionInfo = {
    id:                        string,
    name:                      string,
    runesByPopularity:         Rune[],
    runesByWinRate:            Rune[],
    itemsByPopularity:         Item[],
    itemsByWinRate:            Item[],
    startingItemsByPopularity: Item[],
    startingItemsByWinRate:    Item[],
    role:                      string
} | null

export type Rune = {
    Id:    number,
    Asset: string,
    Info: {
        Name:        string,
        Description: string
    }
}

export type Item = {
    Id:    number,
    Asset: string,
    Name:  string,
}

export type TierListItem = {
    name: string,
    id: number,
    winrate: number,
    tier: 'S' | 'A' | 'B' | 'C' | 'D',
    role: Role,
}

export type Role = 'ALL' | 'TOP' | 'JUNGLE' | 'MID' | 'ADC' | 'SUP';

export type TierList = {
    "ALL": TierListItem[],
    "TOP": TierListItem[],
    "JUNGLE": TierListItem[],
    "MID": TierListItem[],
    "ADC": TierListItem[],
    "SUP": TierListItem[],
}

export let RuneColors  = {
    '8000': '#FDE047',
    '8100': '#DC2626',
    '8200': '#6366F1',
    '8300': '#38BDF8',
    '8400': '#16A34A'
}

export type UserInfo = {
    username: string,
    iconId:   string,
}

export enum EventType {
    USER_INFO         = 0,
    CHAMPION_CHANGE   = 1,
    QUIT_CHAMP_SELECT = 2,
}

export type Event = {
    type:     EventType.USER_INFO;
    username: string;
    iconId:   string;
} | {
    type:  EventType.CHAMPION_CHANGE;
    id:                        string;
    name:                      string;
    runesByPopularity:         Rune[];
    runesByWinRate:            Rune[];
    itemsByPopularity:         Item[];
    itemsByWinRate:            Item[];
    startingItemsByPopularity: Item[];
    startingItemsByWinRate:    Item[];
    role:                      string;
} | {
    type: EventType.QUIT_CHAMP_SELECT;
}

export function championTileURL(championId: string): string {
    return `${CDRAGON}/plugins/rcp-be-lol-game-data/global/default/v1/champion-tiles/${championId}/${championId}000.jpg`;
}