import { Component } from 'solid-js';

const Tooltip: Component<{ text: string }> = (props) => {
    return (
        // Set class "group" on anything that will encapsulate the tooltip
        <span class="transform text-xs whitespace-nowrap absolute top-0 left-1/2 -translate-y-[115%] p-1 -translate-x-1/2
            bg-black/80 invisible group-hover:visible rounded text-white font-normal">
            {props.text}
        </span>
    )
}

export default Tooltip;