import { Rune } from "../types/types";
import Tooltip from "./Tooltip";

function StatComponent(props: {rune: Rune}) {
    return (
        <span class="relative group w-6 h-6 my-1 bg-slate-600 rounded-full flex items-center justify-center">
            <span class="bg-gray-900 w-5 h-5 flex justify-center items-center rounded-full">
                <img src={props.rune.Asset} class="inline w-5 h-5"></img>
            </span>
            <Tooltip text={props.rune.Info.Name} />
        </span>
    )
}

export default StatComponent;