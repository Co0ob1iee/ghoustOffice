import { cva } from "class-variance-authority"
import clsx from "clsx"
const buttonVariants = cva("inline-flex items-center justify-center whitespace-nowrap rounded-xl border px-4 py-2 text-sm transition-colors", {
  variants:{ variant:{ default:"bg-slate-800 border-slate-700 text-slate-100 hover:border-blue-400", ok:"border-green-500/70 hover:border-green-400", danger:"border-red-500/70 hover:border-red-400" } },
  defaultVariants:{ variant:"default" }
})
export function Button({ className, variant, ...props }){
  return <button className={clsx(buttonVariants({variant}), className)} {...props} />
}
