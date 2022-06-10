import { Rune } from "../types/types";

function StatComponent({rune}: {rune: Rune}) {
    return (
        <span class="w-6 h-6 my-1 bg-slate-600 rounded-full flex items-center justify-center">
            <span class="bg-gray-900 w-5 h-5 flex justify-center items-center rounded-full">
                <img src={rune.Asset} class="inline w-5 h-5"></img>
            </span>
        </span>
    )
}

export default StatComponent;