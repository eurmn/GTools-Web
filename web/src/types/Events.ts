export type UserInfo = {
    type:  EventType;
    username: string;
    iconId:   string;
}

export enum EventType {
    USER_INFO = 0,
    LCU_UPDATE = 1
}

export type Event = {
    type: 0;
    username: string;
    iconId:   string;
} | {
    type: EventType.LCU_UPDATE;
}