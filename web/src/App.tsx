import { Component, createSignal, For, Show } from 'solid-js';
import RuneComponent from './components/RuneComponent';
import StatComponent from './components/StatComponent';

import { CDRAGON, Event, EventType, ChampionInfo, UserInfo, Rune } from './types/types'

function iconURL(iconId: string): string {
    return `${CDRAGON}/plugins/rcp-be-lol-game-data/global/default/v1/profile-icons/${iconId}.jpg`;
}

function championTileURL(championId: string): string {
    return `${CDRAGON}/plugins/rcp-be-lol-game-data/global/default/v1/champion-tiles/${championId}/${championId}000.jpg`;
}

function importRunes(runes: Rune[], champion_id: string, role: string) {
    let ids = runes.map(rune => rune.Id)
    fetch('http://' + location.hostname + ':4246/import-runes', {
        method: 'POST',
        body: JSON.stringify({
            runes: ids,
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
                    id:         data.id,
                    name:       data.name,
                    runes:      data.runes,
                    role:       data.role
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

    /* fetch('http://' + location.hostname + ':4246/sample-build').then(res => {
        res.json().then(data => {
            setCurrentChampion(data)
        });
    }).catch(console.log); */

    return (
        <div class="h-full bg-slate-900 text-white py-4 px-4 font-inter flex flex-col">
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
            </Show>
            <Show when={currentChampion()}>
                <div class="h-full flex flex-col justify-center items-center text-gray-200">
                    <div class="flex items-center justify-center text-5xl">
                        <span class="bg-slate-600 h-[1.5em] w-[1.5em] relative rounded-full mx-4">
                            <img src={championTileURL(currentChampion()!.id)} alt={`${currentChampion()!.name} icon`}
                                class="text-center h-[1.4em] w-[1.4em] rounded-full absolute top-1/2 left-1/2
                                        -translate-x-1/2 -translate-y-1/2" />
                        </span>
                        <span class="font-extrabold text-center">{currentChampion()!.name}</span>
                    </div>
                    <div class="flex justify-center flex-wrap max-h-1/2">
                        <span class="p-2 flex flex-col w-1/4 min-w-full md:min-w-min">
                            <img src={currentChampion()!.runes[0].Asset} alt="" class="w-5 h-5 mt-5 mx-auto inline" />
                            <RuneComponent rune={currentChampion()!.runes[2]} background={false} bigger={true} treeId={currentChampion()!.runes[0].Id} />
                            <For each={currentChampion()!.runes.slice(3, 6)}>
                                {(rune) => <RuneComponent rune={rune} treeId={currentChampion()!.runes[0].Id} />}
                            </For>
                        </span>
                        <span class="p-2 flex flex-col w-1/4 min-w-full md:min-w-min">
                            <img src={currentChampion()!.runes[1].Asset} alt="" class="w-5 h-5 mt-5 mx-auto inline" />
                            <For each={currentChampion()!.runes.slice(6, 8)}>
                                {(rune) => <RuneComponent rune={rune} treeId={currentChampion()!.runes[1].Id} />}
                            </For>
                            <span class="flex justify-evenly w-1/4 mx-auto">
                                <For each={currentChampion()!.runes.slice(8)}>
                                    {(rune) => <StatComponent rune={rune} /> }
                                </For>
                            </span>
                        </span>
                    </div>
                    <div class="flex items-center justify-center mt-5 w-full">
                        <span class="py-2 px-5 bg-indigo-700 rounded-md shadow shadow-black/20 hover:cursor-pointer
                                transition-all hover:bg-indigo-800 duration-50 font-bold leading-loose active:scale-95"
                            onClick={() => importRunes(currentChampion()!.runes, currentChampion()!.id, currentChampion()!.role)}>
                            Import Runes
                        </span>
                    </div>
                </div>
            </Show>
        </div>
    );
};

export default App;
