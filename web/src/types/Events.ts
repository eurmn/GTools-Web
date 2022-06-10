export let CDRAGON = 'https://raw.communitydragon.org/latest'

export type ChampionInfo = {
    id:    string,
    name:  string,
    runes: number[],
    role:  string
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
    type:         EventType.CHAMPION_CHANGE;
    championId:   string;
    championName: string;
    runes:        number[];
    role:         string;
}