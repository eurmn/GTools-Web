import { Component, createSignal, Show } from 'solid-js';

import { CDRAGON, Event, EventType, ChampionInfo, UserInfo } from './types/Events'

function iconURL(iconId: string): string {
    return `${CDRAGON}/plugins/rcp-be-lol-game-data/global/default/v1/profile-icons/${iconId}.jpg`;
}

function championTileURL(championId: string): string {
    return `${CDRAGON}/plugins/rcp-be-lol-game-data/global/default/v1/champion-tiles/${championId}/${championId}000.jpg`;
}

function importRunes(runes: number[], champion_id: string, role: string) {
    fetch('http://' + location.hostname + ':4246/import-runes', {
        method: 'POST',
        body: JSON.stringify({
            runes,
            champion_id,
            role
        }),
    }).then(res => {
        console.log(res);
    }).catch(err => {
        console.log(err);
    })
}

const App: Component = () => {
    let [userInfo, setUserInfo] = createSignal<UserInfo>();
    let [currentChampion, setCurrentChampion] = createSignal<ChampionInfo>();

    let ws = new WebSocket('ws://' + location.hostname + ':4246/lcu');
    ws.onmessage = (e: MessageEvent<string>) => {
        let data: Event = JSON.parse(e.data);
        console.log(data);
        switch (data.type) {
            case EventType.USER_INFO:
                setUserInfo({
                    username: data.username,
                    iconId:   data.iconId,
                })
                break;
            case EventType.CHAMPION_CHANGE:
                setCurrentChampion({
                    id:    data.championId,
                    name:  data.championName,
                    runes: data.runes,
                    role:  data.role
                });
                break;
        }
    };

    // close any ws connection before hot-reloading
    if (import.meta.hot) {
        import.meta.hot.on('vite:beforeUpdate', () => {
            ws.close();
        });
    }

    return (
        <div class="h-full w-full bg-slate-900 text-white py-4 px-4 md:px-24 lg:px-72 font-inter">
            <Show when={userInfo()} fallback={
                <div class="text-5xl text-center">Waiting for League Client to Open...</div>
            }>
                <div class="text-xl flex justify-end items-center tracking-tight font-inter antialiased">
                    <span class="p-2 bg-slate-700 rounded-2xl
                        flex items-center shadow shadow-black/50">
                        <img src={iconURL(userInfo()!.iconId)} alt="User Icon" class="rounded-full h-[1.5em] mr-2
                            shadow-inner shadow-black/50" />
                        <span>{userInfo()!.username}</span>
                    </span>
                </div>
                <Show when={currentChampion()}>
                    <div class="flex items-center justify-center text-5xl">
                        <span class="bg-slate-600 h-[1.5em] w-[1.5em] relative rounded-full mx-4">
                            <img src={championTileURL(currentChampion()!.id)} alt={`${currentChampion()!.name} icon`}
                                class="text-center h-[1.4em] w-[1.4em] rounded-full absolute top-1/2 left-1/2
                                    -translate-x-1/2 -translate-y-1/2" />
                        </span>
                        <span class="font-extrabold">{currentChampion()!.name}</span>
                    </div>
                    <div class="flex items-center justify-center mt-5">
                        <span class="py-2 px-5 bg-indigo-700 rounded-md shadow shadow-black/20 hover:cursor-pointer
                            transition-colors hover:bg-indigo-800 duration-50 font-bold leading-loose"
                            onClick={() => importRunes(currentChampion()!.runes, currentChampion()!.id, currentChampion()!.role)}>
                            Import Runes
                        </span>
                    </div>
                </Show>
            </Show>
        </div>
    );
};

export default App;
