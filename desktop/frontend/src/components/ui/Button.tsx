import {ButtonHTMLAttributes} from 'react'

type Props = ButtonHTMLAttributes<HTMLButtonElement> & { variant?: 'primary' | 'secondary', size?: 'sm' | 'md' }

const VARIANT = {
    // Keys inside a primary button take its color instead of looking like separate keycaps.
    primary: 'border-0 bg-primary text-primary-foreground hover:brightness-108 [&_kbd]:bg-transparent [&_kbd]:text-inherit [&_kbd]:border-[color-mix(in_srgb,currentColor_40%,transparent)]',
    secondary: 'border border-border-200 bg-background text-foreground hover:bg-muted',
}
const SIZE = {md: 'h-8 gap-2 px-3 text-sm', sm: 'h-7 gap-1.5 px-2.5 text-xs'}

export function Button({variant = 'secondary', size = 'md', className = '', ...rest}: Props) {
    return <button type="button" className={`inline-flex cursor-pointer items-center rounded-md font-medium whitespace-nowrap ${VARIANT[variant]} ${SIZE[size]} ${className}`} {...rest}/>
}
