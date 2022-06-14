import { Component, createSignal, For } from 'solid-js';
import { CDRAGON, ChampionInfo, Rune } from '../types/types';
import RuneComponent from './RuneComponent';
import StatComponent from './StatComponent';

const RunePageComponent: Component<{ currentChampion: ChampionInfo, winrate: boolean }> = (props) => {
    let [sort, setSort] = createSignal<'runesByWinRate' | 'runesByPopularity'>(
        props.winrate ? 'runesByWinRate' : 'runesByPopularity'
    );

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

    return (
        <div class="flex-1 w-full flex flex-wrap justify-center text-gray-200 md:px-10 content-evenly">
            <div class="flex items-center justify-center text-5xl w-full mb-5 flex-wrap">
                <span class="bg-slate-600 h-[1.5em] w-[1.5em] relative rounded-full">
                    <img src={championTileURL(props.currentChampion.id)} alt={`${props.currentChampion.name} icon`}
                        class="text-center h-[1.4em] w-[1.4em] rounded-full absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2" />
                </span>
                <span class="leading-[1.5em] font-extrabold text-center md:text-5xl text-3xl mx-4">{props.currentChampion.name} {props.currentChampion.role}</span>
                <div class="h-5 flex items-center justify-center px-1 w-full mt-2
                    text-gray-300 font-extrabold text-sm md:text-xs">
                    <span class={`shadow-black/30 rounded-l-full p-1 pl-2 mr-0.5 ${sort() == 'runesByWinRate' ? 'bg-indigo-800 cursor-pointer shadow' : 'bg-slate-700 text-gray-400 shadow-inner' }`}
                        onClick={() => setSort('runesByPopularity')}>Popularity</span>
                    <span class={`shadow-black/30 rounded-r-full p-1 pr-2 ${sort() == 'runesByPopularity' ?'bg-indigo-800 cursor-pointer shadow' : 'bg-slate-700 text-gray-400 shadow-inner' }`}
                        onClick={() => setSort('runesByWinRate')}>Win Rate</span>
                </div>
            </div>
            <span class="p-2 md:w-2/6 w-full flex flex-col max-h-1/5">
                <img src={props.currentChampion[sort()][0].Asset} alt="" class="w-5 h-5 mb-2 mx-auto inline" />
                <RuneComponent rune={props.currentChampion[sort()][2]} background={false} bigger={true} treeId={props.currentChampion[sort()][0].Id} />
                <For each={props.currentChampion[sort()].slice(3, 6)}>
                    {(rune) => <RuneComponent background={true} rune={rune} treeId={props.currentChampion[sort()][0].Id} />}
                </For>
            </span>
            <span class="p-2 md:w-2/6 w-full flex flex-col max-h-2/3">
                <img src={props.currentChampion[sort()][1].Asset} alt="" class="w-5 h-5 mb-2 mx-auto inline" />
                <For each={props.currentChampion[sort()].slice(6, 8)}>
                    {(rune) => <RuneComponent background={true} rune={rune} treeId={props.currentChampion[sort()][1].Id} />}
                </For>
                <span class="flex justify-evenly w-1/2 mx-auto mt-2">
                    <For each={props.currentChampion[sort()].slice(8)}>
                        {(rune) => <StatComponent rune={rune} />}
                    </For>
                </span>
            </span>
            <div class="flex items-center justify-center w-full mt-2 mb-5 max-h-2/3">
                <button class="py-2 px-5 bg-indigo-700 rounded-md shadow shadow-black/20 hover:cursor-pointer
                        transition-all hover:bg-indigo-800 duration-50 font-bold leading-loose active:scale-95"
                    onClick={() => importRunes(props.currentChampion[sort()], props.currentChampion.id, props.currentChampion.role)}>
                    Import Runes
                </button>
            </div>
        </div>
    )
}

export default RunePageComponent;