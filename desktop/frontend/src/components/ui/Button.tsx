import {ButtonHTMLAttributes} from 'react'
import './ui.css'

type Props = ButtonHTMLAttributes<HTMLButtonElement> & { variant?: 'primary' | 'secondary', size?: 'sm' | 'md' }

export function Button({variant = 'secondary', size = 'md', className = '', ...rest}: Props) {
    return <button type="button" className={`btn ${variant} ${size} ${className}`} {...rest}/>
}
