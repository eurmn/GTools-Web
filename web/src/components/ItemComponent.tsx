import { Component } from 'solid-js';
import { Item } from '../types/types';

const ItemComponent: Component<{ item: Item }> = (props) => {
    return (
        <span class="group relative h-8 w-8 bg-slate-700 flex items-center mb-1 justify-center mx-2">
            <img src={props.item.Asset} alt={`${props.item.Name} icon`} class="h-7 w-7 relative"/>
            <span class="text-xs whitespace-nowrap absolute top-0 left-1/2 -translate-y-[115%] p-1 -translate-x-1/2 bg-black/80 invisible group-hover:visible rounded">
                {props.item.Name}
            </span>
        </span>
    )
}

export default ItemComponent;