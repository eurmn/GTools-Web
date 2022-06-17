import { Rune } from "../types/types";

function StatComponent(props: {rune: Rune}) {
    return (
        <span class="relative group w-6 h-6 my-1 bg-slate-600 rounded-full flex items-center justify-center">
            <span class="bg-gray-900 w-5 h-5 flex justify-center items-center rounded-full">
                <img src={props.rune.Asset} class="inline w-5 h-5"></img>
            </span>
            <span class="text-xs whitespace-nowrap absolute top-0 left-1/2 -translate-y-[115%] p-1 -translate-x-1/2 bg-black/50 invisible group-hover:visible rounded">
                {props.rune.Info.Name}
            </span>
        </span>
    )
}

export default StatComponent;