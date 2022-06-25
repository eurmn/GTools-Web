import { Component, createEffect, createSignal, For } from 'solid-js';
import { championTileURL, Role, TierList, TierListItem, CDRAGON } from '../types/types';
import Tooltip from './Tooltip';

function roleIconPath(role: string) {
    let roleStr: string;
    switch (role) {
        case "ADC":
            roleStr = "bottom";
            break;
        case "SUPPORT":
            roleStr = "utility";
            break;
        case "MID":
            roleStr = "middle";
            break;
        default:
            roleStr = role;
            break;
    }
    return CDRAGON + `/plugins/rcp-fe-lol-static-assets/global/default/svg/position-${roleStr.toLowerCase()}-light.svg`;
}

const TierListPageComponent: Component = () => {
    let [tierList, setTierList] = createSignal<TierList>();
    let [roleTierList, setRoleTierList] = createSignal<TierListItem[]>();
    let [currentRole, setCurrentRole] = createSignal<Role>('ALL');

    let tierListColors = {
        S: "text-indigo-600",
        A: "text-green-400",
        B: "text-orange-400",
        C: "text-yellow-300",
        D: "text-red-600",
    }

    createEffect(() => {
        if (tierList()) {
            setRoleTierList(tierList()![currentRole()]);
        }
    });

    fetch('http://' + location.hostname + ':4246/tier-list')
        .then(resp => resp.json().then(j => {
            console.log(j);
            setTierList(j);
        }));

    return (
        <div class="mx-auto w-full md:w-2/3 xl:w-1/2">
            <div class="text-center w-full text-3xl font-inter font-extrabold mb-5">Tier List</div>
            <select class="select bg-slate-800" onchange={(e) =>
                    setCurrentRole(e.currentTarget.value as Role)}>
                <option value="ALL">ALL</option>
                <option value="TOP">TOP</option>
                <option value="JUNGLE">JUNGLE</option>
                <option value="MID">MID</option>
                <option value="ADC">ADC</option>
                <option value="SUP">SUP</option>
            </select>
            <For each={roleTierList()}>
                {(item) =>
                    <div class="flex my-2 items-center p-2 bg-slate-800 rounded shadow-black/30 shadow px-4">
                        <span class={`text-2xl font-inter font-bold mr-2 ${tierListColors[item.tier]}`}>{item.tier}</span>
                        <span class="mx-2 h-11 w-11 bg-slate-500 rounded-full flex items-center justify-center">
                            <img src={championTileURL(item.id.toString())} class="mx-2 h-10 w-10 rounded-full" />
                        </span>
                        <span>{item.name}</span>
                        <span class="ml-auto flex items-center">
                            <img src={roleIconPath(item.role)} alt={item.role} class="opacity-50 inline">
                                <Tooltip text={item.role} />
                            </img>
                            <span class="font-ubuntu text-lg ml-5">{item.winrate.toFixed(2)}<span class="text-xs">% WR</span></span>
                        </span>
                    </div>
                }
            </For>
        </div>
    )
}

export default TierListPageComponent;