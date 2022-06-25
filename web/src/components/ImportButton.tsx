import { Component, Show } from 'solid-js';
import Tooltip from './Tooltip';

const ImportButton: Component<{ action: () => void , text: string, disabled?: boolean}> = (props) => {
    return (
        <div class="flex items-center justify-center mt-2 mb-5 max-h-2/3 mx-1">
            <button disabled={props.disabled} class="py-2 px-5 bg-indigo-700 rounded-md shadow shadow-black/20 hover:cursor-pointer
                    transition-all hover:bg-indigo-800 duration-50 font-bold leading-loose active:scale-95
                    disabled:(bg-gray-700 text-gray-800 cursor-default active:scale-100 transition-none) group relative"
                onClick={() => props.action()}>
                {props.text}
                <Show when={props.disabled}>
                    <Tooltip text='Broken'/>
                </Show>
            </button>
        </div>
    )
}

export default ImportButton;