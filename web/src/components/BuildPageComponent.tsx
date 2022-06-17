import { Component, createEffect, createSignal, For } from 'solid-js';
import { CDRAGON, ChampionInfo, Rune } from '../types/types';
import ItemComponent from './ItemComponent';
import RuneComponent from './RuneComponent';
import StatComponent from './StatComponent';

const BuildPageComponent: Component<{ currentChampion: ChampionInfo }> = (props) => {
    let [sort, setSort] = createSignal<'ByWinRate' | 'ByPopularity'>('ByPopularity');

    let [runeSort, setRuneSort] = createSignal<'runesByWinRate' | 'runesByPopularity'>('runesByPopularity');
    let [itemSort, setItemSort] = createSignal<'itemsByWinRate' | 'itemsByPopularity'>('itemsByPopularity');
    let [startingItemSort, setStartingItemSort] = createSignal<'startingItemsByWinRate' | 'startingItemsByPopularity'>('startingItemsByPopularity');

    createEffect(() => {
        setRuneSort(('runes' + sort()) as 'runesByWinRate' | 'runesByPopularity');
        setItemSort(('items' + sort()) as 'itemsByWinRate' | 'itemsByPopularity');
        setStartingItemSort(('startingItems' + sort()) as 'startingItemsByWinRate' | 'startingItemsByPopularity');
    });

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
            <div class="flex items-center justify-center text-5xl w-full flex-wrap">
                <span class="bg-slate-600 h-[1.5em] w-[1.5em] relative rounded-full">
                    <img src={championTileURL(props.currentChampion.id)} alt={`${props.currentChampion.name} icon`}
                        class="text-center h-[1.4em] w-[1.4em] rounded-full absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2" />
                </span>
                <span class="leading-[1.5em] font-extrabold text-center md:text-5xl text-3xl mx-4">{props.currentChampion.name} {props.currentChampion.role}</span>
            </div>
            <div class="h-5 flex items-center justify-center px-1 w-full my-3 text-gray-300 font-extrabold text-sm md:text-xs">
                <span class={`shadow-black/30 rounded-l-full p-1 pl-2 mr-0.5 ${sort() == 'ByWinRate' ? 'bg-indigo-800 cursor-pointer shadow-md' : 'bg-slate-700 text-gray-400 shadow-inner' }`}
                    onClick={() => setSort('ByPopularity')}>Popularity</span>
                <span class={`shadow-black/30 rounded-r-full p-1 pr-2 ${sort() == 'ByPopularity' ?'bg-indigo-800 cursor-pointer shadow-md' : 'bg-slate-700 text-gray-400 shadow-inner' }`}
                    onClick={() => setSort('ByWinRate')}>Win Rate</span>
            </div>
            <span class="p-2 md:w-2/6 w-full flex flex-col max-h-1/5">
                <img src={props.currentChampion[runeSort()][0].Asset} alt="" class="w-5 h-5 mb-2 mx-auto inline" />
                <RuneComponent rune={props.currentChampion[runeSort()][2]} background={false} bigger={true} treeId={props.currentChampion[runeSort()][0].Id} />
                <For each={props.currentChampion[runeSort()].slice(3, 6)}>
                    {(rune) => <RuneComponent background={true} rune={rune} treeId={props.currentChampion[runeSort()][0].Id} />}
                </For>
            </span>
            <span class="p-2 md:w-2/6 w-full flex flex-col justify-between max-h-2/3 text-gray-300">
                <img src={props.currentChampion[runeSort()][1].Asset} alt="" class="w-5 h-5 mb-2 mx-auto inline" />
                <For each={props.currentChampion[runeSort()].slice(6, 8)}>
                    {(rune) => <RuneComponent background={true} rune={rune} treeId={props.currentChampion[runeSort()][1].Id} />}
                </For>
                <span class="flex justify-evenly w-1/2 mx-auto">
                    <For each={props.currentChampion[runeSort()].slice(8)}>
                        {(rune) => <StatComponent rune={rune} />}
                    </For>
                </span>
                <span class="font-inter leading-tight text-xs">
                    <span class="pl-1">STARTING ITEMS:</span>
                    <span class="flex mt-1 w-full">
                        <For each={props.currentChampion[startingItemSort()]}>
                            {(item) => <ItemComponent item={item} />}
                        </For>
                    </span>
                </span>
                <span class="font-inter leading-tight text-xs">
                    <span class="pl-1">FULL ITEMS:</span>
                    <span class="flex mt-1 w-full">
                        <For each={props.currentChampion[itemSort()]}>
                            {(item) => <ItemComponent item={item} />}
                        </For>
                    </span>
                </span>
            </span>
            <div class="flex items-center justify-center w-full mt-2 mb-5 max-h-2/3">
                <button class="py-2 px-5 bg-indigo-700 rounded-md shadow shadow-black/20 hover:cursor-pointer
                        transition-all hover:bg-indigo-800 duration-50 font-bold leading-loose active:scale-95"
                    onClick={() => importRunes(props.currentChampion[runeSort()], props.currentChampion.id, props.currentChampion.role)}>
                    Import Runes
                </button>
            </div>
        </div>
    )
}

export default BuildPageComponent;