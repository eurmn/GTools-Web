import { Component, createSignal, Show } from 'solid-js';
import BuildPageComponent from './components/BuildPageComponent';
import TierListPageComponent from './components/TierListPageComponent';

import { CDRAGON, Event, EventType, ChampionInfo, UserInfo } from './types/types'

function iconURL(iconId: string): string {
    return `${CDRAGON}/plugins/rcp-be-lol-game-data/global/default/v1/profile-icons/${iconId}.jpg`;
}

const App: Component = () => {
    let [userInfo, setUserInfo] = createSignal<UserInfo>();
    let [currentChampion, setCurrentChampion] = createSignal<ChampionInfo>(null);

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
                    id:                        data.id,
                    name:                      data.name,
                    runesByPopularity:         data.runesByPopularity,
                    runesByWinRate:            data.runesByWinRate,
                    itemsByWinRate:            data.itemsByWinRate,
                    itemsByPopularity:         data.itemsByPopularity,
                    startingItemsByPopularity: data.startingItemsByPopularity,
                    startingItemsByWinRate:    data.startingItemsByWinRate,
                    role:                      data.role
                } as ChampionInfo);
                break;
            case EventType.QUIT_CHAMP_SELECT:
                setCurrentChampion(null)
                break;
        }
    };

    // close any ws connection before hot-reloading
    if (import.meta.hot) {
        import.meta.hot.on('vite:beforeUpdate', () => {
            ws.close();
        });
    }

    // Sample data for Aurelion Sol. Use it to debug the UI without needing
    // to create a custom game.
    /* fetch('http://' + location.hostname + ':4246/sample-build').then(res => {
        res.json().then(data => {
            console.log(data);
            setCurrentChampion({
                id:                        data.id,
                name:                      data.name,
                runesByPopularity:         data.runesByPopularity,
                runesByWinRate:            data.runesByWinRate,
                itemsByWinRate:            data.itemsByWinRate,
                itemsByPopularity:         data.itemsByPopularity,
                startingItemsByPopularity: data.startingItemsByPopularity,
                startingItemsByWinRate:    data.startingItemsByWinRate,
                role:                      data.role
            } as ChampionInfo);
        });
    }).catch(console.log); */

    return (
        <div class="flex flex-col h-full text-white px-4 font-inter">
            <Show when={userInfo()}>
                <div class="my-2 w-full text-xl tracking-tight font-inter antialiased flex">
                    <span class="p-2 bg-slate-700 rounded-2xl ml-auto
                        flex items-center shadow shadow-black/50">
                        <img src={iconURL(userInfo()!.iconId)} alt="User Icon" class="rounded-full h-[1.5em] mr-2
                            shadow-inner shadow-black/50" />
                        <span>{userInfo()!.username}</span>
                    </span>
                </div>
            </Show>
            <Show when={currentChampion()} fallback={
                <TierListPageComponent />
            }>
                <BuildPageComponent currentChampion={currentChampion()!}></BuildPageComponent>
            </Show>
        </div>
    );
};

export default App;
