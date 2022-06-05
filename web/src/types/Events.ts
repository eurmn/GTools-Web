export type UserInfo = {
    type:  EventType;
    username: string;
    iconId:   string;
}

export enum EventType {
    USER_INFO = 0,
    CHAMPION_CHANGE = 1
}

export type Event = {
    type: EventType.USER_INFO;
    username: string;
    iconId: string;
} | {
    type: EventType.CHAMPION_CHANGE;
    championId: string;
}