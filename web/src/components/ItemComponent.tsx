import { Component } from 'solid-js';
import { Item } from '../types/types';
import Tooltip from './Tooltip';

const ItemComponent: Component<{ item: Item }> = (props) => {
    return (
        <span class="group relative h-11 w-11 md:h-8 md:w-8 bg-slate-700 flex items-center mb-1 justify-center mx-1">
            <img src={props.item.Asset} alt={`${props.item.Name} icon`} class="h-10 w-10 md:h-7 md:w-7 relative"/>
            <Tooltip text={props.item.Name} />
        </span>
    )
}

export default ItemComponent;