import { Component, createSignal } from 'solid-js';

import logo from './logo.svg';
import io from 'socket.io-client';
import * as events from './types/Events'

const App: Component = () => {
  let [username, setUsername] = createSignal('');
  let [iconId, setIconId] = createSignal('');
  let [currentChampion, setCurrentChampion] = createSignal('');

  let ws = new WebSocket('ws://' + location.hostname + ':4246/lcu');
  ws.onmessage = (e: MessageEvent<string>) => {
    let data: events.Event = JSON.parse(e.data);
    switch (data.type) {
      case events.EventType.USER_INFO:
        setUsername(data.username);
        setIconId(data.iconId);
        break;
      case events.EventType.CHAMPION_CHANGE:
        setCurrentChampion(data.championId);
    }
  };

  return (
    <div class="h-full w-full bg-slate-900 text-white">
      <span class="text-5xl">{username()}</span>
      <span class="text-3xl">{currentChampion()}</span>
    </div>
  );
};

export default App;
