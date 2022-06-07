import { Component, createSignal, Show } from 'solid-js';

import logo from './logo.svg';
import * as events from './types/Events'

const App: Component = () => {
    let [username, setUsername] = createSignal('');
    let [iconId, setIconId] = createSignal('');
    let [currentChampion, setCurrentChampion] = createSignal('');

    let ws = new WebSocket('ws://' + location.hostname + ':4246/lcu');
    ws.onmessage = (e: MessageEvent<string>) => {
        let data: events.Event = JSON.parse(e.data);
        console.log(data);
        switch (data.type) {
            case events.EventType.USER_INFO:
                setUsername(data.username);
                setIconId(data.iconId);
                break;
            case events.EventType.CHAMPION_CHANGE:
                setCurrentChampion(data.championId);
        }
    };

    window.onclose = () => {
        ws.close();
    }

    return (
        <div class="h-full w-full bg-slate-900 text-white p-4 font-inter">
            <Show when={username()} fallback={
                <div class="text-5xl text-center">Waiting for League Client to Open...</div>
            }>
                    <div class="text-5xl text-center">{username()}</div>
                    <div class="text-3xl text-center">{currentChampion()}</div>
            </Show>
        </div>
    );
};

export default App;
