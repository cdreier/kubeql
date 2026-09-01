import "./FavoriteStar.css";

function StarIcon({ filled }: { filled: boolean }) {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" aria-hidden="true">
      <path
        d="M12 17.27 18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z"
        fill={filled ? "currentColor" : "none"}
        stroke="currentColor"
        strokeWidth="1.6"
        strokeLinejoin="round"
      />
    </svg>
  );
}

type FavoriteStarProps = {
  name: string;
  favorited: boolean;
  onToggle: (name: string) => void;
};

export function FavoriteStar({ name, favorited, onToggle }: FavoriteStarProps) {
  return (
    <button
      type="button"
      className="fav-btn"
      aria-pressed={favorited}
      aria-label={
        favorited
          ? `Remove ${name} from favorites`
          : `Add ${name} to favorites`
      }
      onClick={() => onToggle(name)}
    >
      <StarIcon filled={favorited} />
    </button>
  );
}
