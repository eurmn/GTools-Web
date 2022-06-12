import { classList } from "solid-js/web";
import { Rune } from "../types/types";

function RuneComponent({ rune, bigger = false, treeId, background = true }:
    { rune: Rune, treeId?: number, bigger?: boolean, background?: boolean }) {

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

    return (
        <span class={`flex my-2 items-center`}>
            <span class={`aspect-square ${bigger ? 'h-20 w-20' : 'h-10 w-10'} my-2 rounded-full flex items-center justify-center`}
                style={background ? `background: ${color}` : ''}>
                <img src={rune.Asset} class={`inline ${bigger ? 'h-18 w-18 -translate-x-5' : 'h-9 w-9'} aspect-square`}></img>
            </span>
            <span class={`ml-5 ${bigger ? '-translate-x-10' : ''}`}>
                <div style={`color: ${color}`}>{rune.Info.Name}</div>
                <div class="text-xs text-gray-300">{rune.Info.Description}</div>
            </span>
        </span>
    )
}

export default RuneComponent;