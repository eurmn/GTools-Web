import { createSignal } from "solid-js";
import { Rune } from "../types/types";

function color(treeId: number): string {
    let color: string;
    switch (treeId) {
        case 8000:
            color = '#FDE047';
            break;
        case 8100:
            color = '#DC2626';
            break;
        case 8200:
            color = '#6366F1';
            break;
        case 8300:
            color = '#38BDF8';
            break;
        case 8400:
            color = '#16A34A';
            break;
        default:
            color = 'transparent';
            break;
    }

    return color;
}

function RuneComponent(props: { rune: Rune, treeId?: number, bigger?: boolean, background: boolean }) {
    return (
        <span class={`flex my-2 items-center`}>
            <span class={`aspect-square ${props.bigger ? 'h-20 w-20' : 'h-10 w-10'} my-2 rounded-full flex items-center justify-center`}
                style={props.background ? `background: ${color(props.treeId || 0)}` : ''}>
                <img src={props.rune.Asset} class={`inline ${props.bigger ? 'h-18 w-18 -translate-x-5' : 'h-9 w-9'} aspect-square`}></img>
            </span>
            <span class={`ml-5 ${props.bigger ? '-translate-x-10' : ''} max-h-14 overflow-hidden`}>
                <div style={`color: ${color(props.treeId || 0)}`}>{props.rune.Info.Name}</div>
                <div class="text-xs text-gray-300">{props.rune.Info.Description}</div>
            </span>
        </span>
    )
}

export default RuneComponent;