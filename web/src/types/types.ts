export let CDRAGON = 'https://raw.communitydragon.org/latest'

export type ChampionInfo = {
    id:         string,
    name:       string,
    runes:      Rune[],
    role:       string
}

export type Rune = {
    Id:    number,
    Asset: string,
    Info: {
        Name:        string,
        Description: string
    }
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
    USER_INFO        = 0,
    CHAMPION_CHANGE  = 1,
}

export type Event = {
    type:     EventType.USER_INFO;
    username: string;
    iconId:   string;
} | {
    type:  EventType.CHAMPION_CHANGE;
    id:    string;
    name:  string;
    runes: Rune[];
    role:  string;
}